/*
Copyright © 2023 Thomas von Dein

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package api

import (
	//"github.com/alecthomas/repr"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"

	"bytes"
	"html/template"
	"regexp"
	"strings"
	"time"
)

/*
   Validate a fied by untainting it, modifies field value inplace.
*/
func untaintField(c *fiber.Ctx, orig *string, r *regexp.Regexp, caption string) error {
	if len(*orig) != 0 {
		nt, err := common.Untaint(*orig, r)
		if err != nil {
			return JsonStatus(c, fiber.StatusForbidden,
				"Invalid "+caption+": "+err.Error())
		}
		*orig = nt
	}

	return nil
}

func FormCreate(c *fiber.Ctx, cfg *cfg.Config, db *Db) error {
	id := uuid.NewString()

	var formdata common.Form

	// init form obj
	entry := &common.Form{Id: id, Created: common.Timestamp{Time: time.Now()}, Type: common.TypeForm}

	// retrieve the API Context name from the session
	apicontext, err := SessionGetApicontext(c)
	if err != nil {
		return JsonStatus(c, fiber.StatusInternalServerError,
			"Unable to initialize session store from context: "+err.Error())
	}
	entry.Context = apicontext

	// extract auxiliary form data (expire field et al)
	if err := c.BodyParser(&formdata); err != nil {
		return JsonStatus(c, fiber.StatusInternalServerError,
			"bodyparser error : "+err.Error())
	}

	// post process inputdata
	if len(formdata.Expire) == 0 {
		entry.Expire = "asap"
	} else {
		if err := untaintField(c, &formdata.Expire, cfg.RegDuration, "expire data"); err != nil {
			return err
		}
		entry.Expire = formdata.Expire
	}

	if err := untaintField(c, &formdata.Notify, cfg.RegDuration, "email address"); err != nil {
		return err
	}
	entry.Notify = formdata.Notify

	if err := untaintField(c, &formdata.Description, cfg.RegDuration, "description"); err != nil {
		return err
	}
	entry.Description = formdata.Description

	// get url [and zip if there are multiple files]
	returnUrl := strings.Join([]string{cfg.Url, "form", id}, "/")
	entry.Url = returnUrl

	Log("Now serving %s", returnUrl)
	Log("Expire set to: %s", entry.Expire)
	Log("Form created with API-Context %s", entry.Context)

	// we do this in the background to not thwart the server
	go func() {
		if err := db.Insert(id, entry); err != nil {
			Log("Failed to insert: " + err.Error())
		}
	}()

	// everything went well so far
	res := &common.Response{Forms: []*common.Form{entry}}
	res.Success = true
	res.Code = fiber.StatusOK
	return c.Status(fiber.StatusOK).JSON(res)
}

// delete form
func FormDelete(c *fiber.Ctx, cfg *cfg.Config, db *Db) error {
	id, err := common.Untaint(c.Params("id"), cfg.RegKey)
	if err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Invalid id provided!")
	}

	if len(id) == 0 {
		return JsonStatus(c, fiber.StatusForbidden,
			"No id specified!")
	}

	// retrieve the API Context name from the session
	apicontext, err := SessionGetApicontext(c)
	if err != nil {
		return JsonStatus(c, fiber.StatusInternalServerError,
			"Unable to initialize session store from context: "+err.Error())
	}

	err = db.Delete(apicontext, id)
	if err != nil {
		// non existent db entry with that id, or other db error, see logs
		return JsonStatus(c, fiber.StatusForbidden,
			"No form with that id could be found!")
	}

	return nil
}

// returns the whole list + error code, no post processing by server
func FormsList(c *fiber.Ctx, cfg *cfg.Config, db *Db) error {
	// fetch filter from body(json expected)
	setcontext := new(SetContext)
	if err := c.BodyParser(setcontext); err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Unable to parse body: "+err.Error())
	}

	filter, err := common.Untaint(setcontext.Apicontext, cfg.RegKey)
	if err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Invalid api context filter provided!")
	}

	query, err := common.Untaint(setcontext.Query, cfg.RegQuery)
	if err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Invalid query provided!")
	}

	// retrieve the API Context name from the session
	apicontext, err := SessionGetApicontext(c)
	if err != nil {
		return JsonStatus(c, fiber.StatusInternalServerError,
			"Unable to initialize session store from context: "+err.Error())
	}

	// get list
	response, err := db.List(apicontext, filter, query, common.TypeForm)
	if err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Unable to list forms: "+err.Error())
	}

	// if we reached this point we can signal success
	response.Success = true
	response.Code = fiber.StatusOK

	return c.Status(fiber.StatusOK).JSON(response)
}

// returns just one form obj + error code
func FormDescribe(c *fiber.Ctx, cfg *cfg.Config, db *Db) error {
	id, err := common.Untaint(c.Params("id"), cfg.RegKey)
	if err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Invalid id provided!")
	}

	// retrieve the API Context name from the session
	apicontext, err := SessionGetApicontext(c)
	if err != nil {
		return JsonStatus(c, fiber.StatusInternalServerError,
			"Unable to initialize session store from context: "+err.Error())
	}

	response, err := db.Get(apicontext, id, common.TypeForm)
	if err != nil || len(response.Forms) == 0 {
		return JsonStatus(c, fiber.StatusForbidden,
			"No form with that id could be found!")
	}

	for _, form := range response.Forms {
		form.Url = strings.Join([]string{cfg.Url, "form", id}, "/")
	}

	// if we reached this point we can signal success
	response.Success = true
	response.Code = fiber.StatusOK

	return c.Status(fiber.StatusOK).JSON(response)
}

/*
   Render the upload  html form. Template given  by --formpage, stored
   as  text  in  cfg.Formpage.  It will  be  rendered  using  golang's
   template engine,  data to  be filled  in is  the form  matching the
   given id.
*/
func FormPage(c *fiber.Ctx, cfg *cfg.Config, db *Db, shallexpire bool) error {
	id, err := common.Untaint(c.Params("id"), cfg.RegKey)
	if err != nil {
		return c.Status(fiber.StatusForbidden).SendString("Invalid id provided!")
	}

	apicontext, err := SessionGetApicontext(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			SendString("Unable to initialize session store from context:" + err.Error())
	}

	response, err := db.Get(apicontext, id, common.TypeForm)
	if err != nil || len(response.Forms) == 0 {
		return c.Status(fiber.StatusForbidden).
			SendString("No form with that id could be found!")
	}

	t := template.New("form")
	if t, err = t.Parse(cfg.Formpage); err != nil {
		return c.Status(fiber.StatusInternalServerError).
			SendString("Unable to load form template: " + err.Error())
	}

	// prepare upload url
	uploadurl := strings.Join([]string{cfg.ApiPrefix + ApiVersion, "uploads"}, "/")
	response.Forms[0].Url = uploadurl

	var out bytes.Buffer
	if err := t.Execute(&out, response.Forms[0]); err != nil {
		return c.Status(fiber.StatusInternalServerError).
			SendString("Unable to render form template: " + err.Error())
	}

	c.Set("Content-type", "text/html; charset=utf-8")
	return c.Status(fiber.StatusOK).SendString(out.String())
}

func FormModify(c *fiber.Ctx, cfg *cfg.Config, db *Db) error {
	var formdata common.Form

	// retrieve the API Context name from the session
	apicontext, err := SessionGetApicontext(c)
	if err != nil {
		return JsonStatus(c, fiber.StatusInternalServerError,
			"Unable to initialize session store from context: "+err.Error())
	}

	id, err := common.Untaint(c.Params("id"), cfg.RegKey)
	if err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Invalid id provided!")
	}

	// extract form data
	if err := c.BodyParser(&formdata); err != nil {
		return JsonStatus(c, fiber.StatusInternalServerError,
			"bodyparser error : "+err.Error())
	}

	// post process input data
	if err := untaintField(c, &formdata.Expire, cfg.RegDuration, "expire data"); err != nil {
		return err
	}

	if err := untaintField(c, &formdata.Notify, cfg.RegDuration, "email address"); err != nil {
		return err
	}

	if err := untaintField(c, &formdata.Description, cfg.RegDuration, "description"); err != nil {
		return err
	}

	// lookup orig entry
	response, err := db.Get(apicontext, id, common.TypeForm)
	if err != nil || len(response.Forms) == 0 {
		return JsonStatus(c, fiber.StatusForbidden,
			"No form with that id could be found!")
	}

	form := response.Forms[0]

	// modify fields
	if formdata.Expire != "" {
		form.Expire = formdata.Expire
	}

	if formdata.Notify != "" {
		form.Notify = formdata.Notify
	}

	if formdata.Description != "" {
		form.Description = formdata.Description
	}

	// run in foreground because we need the feedback here
	if err := db.Insert(id, form); err != nil {
		return JsonStatus(c, fiber.StatusForbidden,
			"Failed to insert: "+err.Error())
	}

	res := &common.Response{Forms: []*common.Form{form}}
	res.Success = true
	res.Code = fiber.StatusOK
	return c.Status(fiber.StatusOK).JSON(res)
}
