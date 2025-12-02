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
	"fmt"
	"time"

	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"
	"github.com/gofiber/fiber/v2"
)

const ApiVersion string = "/v1"

// Binding from JSON, data coming from user, not tainted
type Meta struct {
	Expire string `json:"expire" form:"expire"`
}

// incoming id
type Id struct {
	Id string `json:"name" xml:"name" form:"name"`
}

// vaious helbers
func Log(format string, values ...any) {
	fmt.Printf("[DEBUG] "+format+"\n", values...)
}

func Ts() string {
	t := time.Now()
	return t.Format("2006-01-02-15-04-")
}

/*
Retrieve the  API Context  name from the  session, assuming  is has
been  successfully  authenticated. However,  if  there  are no  api
contexts     defined,    we'll     use     'default'    (set     in
auth.validateAPIKey()).

If there's no apicontext in the session, assume unauth user, return ""
*/
func SessionGetApicontext(c *fiber.Ctx) (string, error) {
	sess, err := Sessionstore.Get(c)
	if err != nil {
		return "", fmt.Errorf("Unable to initialize session store from context: %s", err.Error())
	}

	apicontext := sess.Get("apicontext")
	if apicontext != nil {
		return apicontext.(string), nil
	}

	return "", nil
}

/*
Retrieve the formid  (aka onetime api key) from the  session. It is
configured if an upload request has been successfully authenticated
using a onetime key.
*/
func SessionGetFormId(c *fiber.Ctx) (string, error) {
	sess, err := Sessionstore.Get(c)
	if err != nil {
		return "", fmt.Errorf("Unable to initialize session store from context: %s", err.Error())
	}

	formid := sess.Get("formid")
	if formid != nil {
		return formid.(string), nil
	}

	return "", nil
}

/*
	Calculate   if  time   is   up  based   on   start  time.Time   and
	duration. Returns  true if time  is expired. Start time  comes from
	the database.

aka:

	if(now - start) >= duration { time is up}
*/
func IsExpired(conf *cfg.Config, start time.Time, duration string) bool {
	var expiretime int // seconds

	now := time.Now()

	if duration == "asap" {
		expiretime = conf.DefaultExpire
	} else {
		expiretime = common.Duration2int(duration)
	}

	if now.Unix()-start.Unix() >= int64(expiretime) {
		return true
	}

	return false
}
