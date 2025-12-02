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
package lib

import (
	"encoding/json"
	"errors"
	"fmt"

	//"github.com/alecthomas/repr"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"codeberg.org/scip/ephemerup/common"
	"codeberg.org/scip/ephemerup/upctl/cfg"
	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/schollz/progressbar/v3"
)

type Response struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Request struct {
	R   *req.Request
	Url string
}

type ListParams struct {
	Apicontext string `json:"apicontext"`
	Query      string `json:"query"`
}

const Maxwidth = 12

/*
Create a new request object for outgoing queries
*/
func Setup(c *cfg.Config, path string) *Request {
	client := req.C()
	if c.Debug {
		client.DevMode()
	}

	client.SetUserAgent("upctl-" + cfg.VERSION)

	R := client.R()

	if c.Retries > 0 {
		// Enable retry and set the maximum retry count.
		R.SetRetryCount(c.Retries).
			//  Set  the  retry  sleep   interval  with  a  commonly  used
			//   algorithm:  capped   exponential   backoff  with   jitter
			// (https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/).
			SetRetryBackoffInterval(1*time.Second, 5*time.Second).
			AddRetryHook(func(resp *req.Response, err error) {
				req := resp.Request.RawRequest
				if c.Debug {
					fmt.Println("Retrying endpoint request:", req.Method, req.URL, err)
				}
			})
	}

	if len(c.Apikey) > 0 {
		client.SetCommonBearerAuthToken(c.Apikey)
	}

	if c.Mock {
		// intercept, used by unit tests
		httpmock.ActivateNonDefault(client.GetClient())
	}

	return &Request{Url: c.Endpoint + path, R: R}
}

/*
Iterate over args, considering the  elements are filenames, and add
them to the request.
*/
func GatherFiles(rq *Request, args []string) error {
	for _, file := range args {
		info, err := os.Stat(file)

		if os.IsNotExist(err) {
			return err
		}

		if info.IsDir() {
			err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !info.IsDir() {
					rq.R.SetFile("upload[]", path)
				}
				return nil
			})

			if err != nil {
				return err
			}
		} else {
			rq.R.SetFile("upload[]", file)
		}
	}

	return nil
}

/*
Check  HTTP  Response Code  and  validate  JSON status  output,  if
any. Turns'em into a regular error
*/
func HandleResponse(c *cfg.Config, resp *req.Response) error {
	// we expect a json response, extract the error, if any
	r := Response{}

	if c.Debug {
		trace := resp.Request.TraceInfo()
		fmt.Println(trace.Blame())
		fmt.Println("----------")
		fmt.Println(trace)
	}

	if err := json.Unmarshal([]byte(resp.String()), &r); err != nil {
		// text output!
		r.Message = resp.String()
	}

	if !resp.IsSuccessState() {
		return fmt.Errorf("bad response: %s (%s)", resp.Status, r.Message)
	}

	if !r.Success {
		if len(r.Message) == 0 {
			if resp.Err != nil {
				return resp.Err
			} else {
				return errors.New("Unknown error")
			}
		} else {
			return errors.New(r.Message)
		}
	}

	return nil
}

func UploadFiles(w io.Writer, c *cfg.Config, args []string) error {
	// setup url, req.Request, timeout handling etc
	rq := Setup(c, "/uploads")

	// collect files to upload from @argv
	if err := GatherFiles(rq, args); err != nil {
		return err
	}

	if !c.Silent {
		// progres bar
		bar := progressbar.Default(100)
		var left float64
		rq.R.SetUploadCallbackWithInterval(func(info req.UploadInfo) {
			left = float64(info.UploadedSize) / float64(info.FileSize) * 100.0
			if err := bar.Add(int(left)); err != nil {
				fmt.Print("\r")
			}
		}, 10*time.Millisecond)
	}

	// actual post w/ settings
	resp, err := rq.R.
		SetFormData(map[string]string{
			"expire":      c.Expire,
			"description": c.Description,
		}).
		Post(rq.Url)

	if err != nil {
		return err
	}

	if err := HandleResponse(c, resp); err != nil {
		return err
	}

	return RespondExtended(w, resp)
}

func List(w io.Writer, c *cfg.Config, args []string, typ int) error {
	var rq *Request

	switch typ {
	case common.TypeUpload:
		rq = Setup(c, "/uploads")
	case common.TypeForm:
		rq = Setup(c, "/forms")
	}

	params := &ListParams{Apicontext: c.Apicontext, Query: c.Query}
	resp, err := rq.R.
		SetBodyJsonMarshal(params).
		Get(rq.Url)

	if err != nil {
		return err
	}

	if err := HandleResponse(c, resp); err != nil {
		return err
	}

	switch typ {
	case common.TypeUpload:
		return UploadsRespondTable(w, resp)
	case common.TypeForm:
		return FormsRespondTable(w, resp)
	}

	return nil
}

func Delete(w io.Writer, c *cfg.Config, args []string, typ int) error {
	for _, id := range args {
		var rq *Request
		caption := "Upload"

		switch typ {
		case common.TypeUpload:
			rq = Setup(c, "/uploads/"+id)
		case common.TypeForm:
			rq = Setup(c, "/forms/"+id)
			caption = "Form"
		}

		resp, err := rq.R.Delete(rq.Url)

		if err != nil {
			return err
		}

		if err := HandleResponse(c, resp); err != nil {
			return err
		}

		fmt.Fprintf(w, "%s %s successfully deleted.\n", caption, id)
	}

	return nil
}

func Describe(w io.Writer, c *cfg.Config, args []string, typ int) error {
	if len(args) == 0 {
		return errors.New("No id provided!")
	}

	var rq *Request
	id := args[0] // we describe only 1 object

	switch typ {
	case common.TypeUpload:
		rq = Setup(c, "/uploads/"+id)
	case common.TypeForm:
		rq = Setup(c, "/forms/"+id)
	}

	resp, err := rq.R.Get(rq.Url)

	if err != nil {
		return err
	}

	if err := HandleResponse(c, resp); err != nil {
		return err
	}

	return RespondExtended(w, resp)
}

func Download(w io.Writer, c *cfg.Config, args []string) error {
	if len(args) == 0 {
		return errors.New("No id provided!")
	}

	id := args[0]

	rq := Setup(c, "/uploads/"+id+"/file")

	if !c.Silent {
		// progres bar
		bar := progressbar.Default(100)

		callback := func(info req.DownloadInfo) {
			if info.Response.Response != nil {
				if err := bar.Add(1); err != nil {
					fmt.Print("\r")
				}
			}
		}

		rq.R.SetDownloadCallback(callback)
	}

	resp, err := rq.R.
		SetOutputFile(id).
		Get(rq.Url)

	if err != nil {
		return err
	}

	if !resp.IsSuccessState() {
		return fmt.Errorf("bad response: %s", resp.Status)
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		os.Remove(id)
		return err
	}

	filename := params["filename"]
	if filename == "" {
		os.Remove(id)
		return fmt.Errorf("No filename provided!")
	}

	cleanfilename, _ := common.Untaint(filename, regexp.MustCompile(`[^a-zA-Z0-9\-\._]`))

	if err := os.Rename(id, cleanfilename); err != nil {
		os.Remove(id)
		return fmt.Errorf("\nUnable to rename file: %s", err.Error())
	}

	fmt.Fprintf(w, "%s successfully downloaded to file %s.", id, cleanfilename)

	return nil
}

func Modify(w io.Writer, c *cfg.Config, args []string, typ int) error {
	id := args[0]
	var rq *Request

	// setup url, req.Request, timeout handling etc
	switch typ {
	case common.TypeUpload:
		rq = Setup(c, "/uploads/"+id)
		rq.R.
			SetBody(&common.Upload{
				Expire:      c.Expire,
				Description: c.Description,
			})
	case common.TypeForm:
		rq = Setup(c, "/forms/"+id)
		rq.R.
			SetBody(&common.Form{
				Expire:      c.Expire,
				Description: c.Description,
				Notify:      c.Notify,
			})
	}

	// actual put w/ settings
	resp, err := rq.R.Put(rq.Url)

	if err != nil {
		return err
	}

	if err := HandleResponse(c, resp); err != nil {
		return err
	}

	return RespondExtended(w, resp)
}

/**** Forms stuff ****/
func CreateForm(w io.Writer, c *cfg.Config) error {
	// setup url, req.Request, timeout handling etc
	rq := Setup(c, "/forms")

	// actual post w/ settings
	resp, err := rq.R.
		SetBody(&common.Form{
			Expire:      c.Expire,
			Description: c.Description,
			Notify:      c.Notify,
		}).
		Post(rq.Url)

	if err != nil {
		return err
	}

	if err := HandleResponse(c, resp); err != nil {
		return err
	}

	return RespondExtended(w, resp)
}
