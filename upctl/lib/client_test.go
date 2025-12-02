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
	//"github.com/alecthomas/repr"
	"bytes"
	"fmt"
	"github.com/jarcoal/httpmock"
	"codeberg.org/scip/ephemerup/common"
	"codeberg.org/scip/ephemerup/upctl/cfg"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const endpoint string = "http://localhost:8080/v1"

type Unit struct {
	name     string
	apikey   string   // set to something else than "token" to fail auth
	wantfail bool     // true: expect to fail
	files    []string // path relative to ./t/
	expect   string   // regex used to parse the output

	sendcode int    // for httpmock
	sendjson string // struct to respond with
	sendfile string // bare file content to be sent
	route    string // dito
	method   string // method to use
}

// simulate our ephemerup server
func Intercept(tt Unit) {
	httpmock.RegisterResponder(tt.method, endpoint+tt.route,
		func(request *http.Request) (*http.Response, error) {
			var resp *http.Response

			if tt.sendfile != "" {
				// simulate a file download
				content, err := ioutil.ReadFile(tt.sendfile)
				if err != nil {
					panic(err) // should not happen
				}

				stat, err := os.Stat(tt.sendfile)
				if err != nil {
					panic(err) // should not happen as well
				}

				resp = httpmock.NewStringResponse(tt.sendcode, string(content))
				resp.Header.Set("Content-Type", "text/markdown; charset=utf-8")
				resp.Header.Set("Content-Length", strconv.Itoa(int(stat.Size())))
				resp.Header.Set("Content-Disposition", "attachment; filename='t1'")
			} else {
				// simulate JSON response
				resp = httpmock.NewStringResponse(tt.sendcode, tt.sendjson)
				resp.Header.Set("Content-Type", "application/json; charset=utf-8")
			}

			return resp, nil
		})
}

// execute the actual test
func Check(t *testing.T, tt Unit, w *bytes.Buffer, err error) {
	testname := fmt.Sprintf("%s-%t", tt.name, tt.wantfail)

	if err != nil && !tt.wantfail {
		t.Errorf("%s failed! wantfail: %t, error: %s", testname, tt.wantfail, err.Error())
	}

	if tt.expect != "" {
		got := strings.TrimSpace(w.String())
		r := regexp.MustCompile(tt.expect)
		if !r.MatchString(got) {
			t.Errorf("%s failed! error: output does not match!\nexpect: %s\ngot:\n%s", testname, tt.expect, got)
		}
	}
}

func TestUploadFiles(t *testing.T) {
	conf := &cfg.Config{
		Mock:     true,
		Apikey:   "token",
		Endpoint: endpoint,
		Silent:   true,
	}

	tests := []Unit{
		{
			name:     "upload-file",
			apikey:   "token",
			wantfail: false,
			route:    "/uploads",
			sendcode: 200,
			sendjson: `{"success": true}`,
			files:    []string{"../t/t1"}, // pwd is lib/ !
			method:   "POST",
		},
		{
			name:     "upload-dir",
			apikey:   "token",
			wantfail: false,
			route:    "/uploads",
			sendcode: 200,
			sendjson: `{"success": true}`,
			files:    []string{"../t"}, // pwd is lib/ !
			method:   "POST",
		},
		{
			name:     "upload-catch-nonexistent-file",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads",
			sendcode: 200,
			sendjson: `{"success": false}`,
			files:    []string{"../t/none"},
			method:   "POST",
		},
		{
			name:     "upload-catch-no-access",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads",
			sendcode: 403,
			sendjson: `{"success": false}`,
			files:    []string{"../t/t1"},
			method:   "POST",
		},
		{
			name:     "upload-check-output",
			apikey:   "token",
			wantfail: false,
			route:    "/uploads",
			sendcode: 200,
			sendjson: `{"uploads":[
                           {
                              "id":"cc2c965a","expire":"asap","file":"t1","members":["t1"],
                              "uploaded":1679396814.890502,"context":"foo","url":""
                           }
                       ],
                       "success":true,
                       "message":"Download url: http://localhost:8080/download/cc2c965a/t1",
                       "code":200}`,
			files:  []string{"../t/t1"}, // pwd is lib/ !
			method: "POST",
			expect: "Expire: On first access",
		},
	}

	for _, unit := range tests {
		var w bytes.Buffer
		Intercept(unit)
		Check(t, unit, &w, UploadFiles(&w, conf, unit.files))
	}
}

func TestList(t *testing.T) {
	conf := &cfg.Config{
		Mock:     true,
		Apikey:   "token",
		Endpoint: endpoint,
		Silent:   true,
	}

	listing := `{"uploads":[
                         {
                              "id":"cc2c965a","expire":"asap","file":"t1","members":["t1"],
                              "uploaded":1679396814.890502,"context":"foo","url":""
                         }
                       ],
                       "success":true,
                       "message":"",
                       "code":200}`

	listingnoaccess := `{"success":false,"message":"invalid context","code":503}`

	tests := []Unit{
		{
			name:     "list",
			apikey:   "token",
			wantfail: false,
			route:    "/uploads",
			sendcode: 200,
			sendjson: listing,
			files:    []string{},
			method:   "GET",
			expect:   `cc2c965a\s*asap\s*foo\s*2023-03-21`, // expect tabular output
		},
		{
			name:     "list-catch-empty-json",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads",
			sendcode: 404,
			sendjson: "",
			files:    []string{},
			method:   "GET",
		},
		{
			name:     "list-catch-no-access",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads",
			sendcode: 503,
			sendjson: listingnoaccess,
			files:    []string{},
			method:   "GET",
		},
	}

	for _, unit := range tests {
		var w bytes.Buffer
		Intercept(unit)
		Check(t, unit, &w, List(&w, conf, []string{}, common.TypeUpload))
	}
}

func TestDescribe(t *testing.T) {
	conf := &cfg.Config{
		Mock:     true,
		Apikey:   "token",
		Endpoint: endpoint,
		Silent:   true,
	}

	listing := `{"uploads":[
                         {
                              "id":"cc2c965a","expire":"asap","file":"t1","members":["t1"],
                              "uploaded":1679396814.890502,"context":"foo","url":""
                         }
                       ],
                       "success":true,
                       "message":"",
                       "code":200}`

	listingnoaccess := `{"success":false,"message":"invalid context","code":503}`

	tests := []Unit{
		{
			name:     "describe",
			apikey:   "token",
			wantfail: false,
			route:    "/uploads/",
			sendcode: 200,
			sendjson: listing,
			files:    []string{"cc2c965a"},
			method:   "GET",
			expect:   `Created: 2023-03-21`,
		},
		{
			name:     "describe-catch-empty-json",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads/",
			sendcode: 200,
			sendjson: "",
			files:    []string{"cc2c965a"},
			method:   "GET",
		},
		{
			name:     "describe-catch-no-access",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads/",
			sendcode: 503,
			sendjson: listingnoaccess,
			files:    []string{"cc2c965a"},
			method:   "GET",
		},
	}

	for _, unit := range tests {
		var w bytes.Buffer
		unit.route += unit.files[0]
		Intercept(unit)
		Check(t, unit, &w, Describe(&w, conf, unit.files, common.TypeUpload))
	}
}

func TestDelete(t *testing.T) {
	conf := &cfg.Config{
		Mock:     true,
		Apikey:   "token",
		Endpoint: endpoint,
		Silent:   true,
	}

	listingnoaccess := `{"success":false,"message":"invalid context","code":503}`

	tests := []Unit{
		{
			name:     "delete",
			apikey:   "token",
			wantfail: false,
			route:    "/uploads/",
			sendcode: 200,
			sendjson: `{"success":true,"message":"","code":200}`,
			files:    []string{"cc2c965a"},
			method:   "DELETE",
			expect:   `Upload cc2c965a successfully deleted`,
		},
		{
			name:     "delete-catch-empty-json",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads/",
			sendcode: 200,
			sendjson: "",
			files:    []string{"cc2c965a"},
			method:   "DELETE",
		},
		{
			name:     "delete-catch-no-access",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads/",
			sendcode: 503,
			sendjson: listingnoaccess,
			files:    []string{"cc2c965a"},
			method:   "DELETE",
		},
	}

	for _, unit := range tests {
		var w bytes.Buffer
		unit.route += unit.files[0]
		Intercept(unit)
		Check(t, unit, &w, Delete(&w, conf, unit.files, common.TypeUpload))
	}
}

func TestDownload(t *testing.T) {
	conf := &cfg.Config{
		Mock:     true,
		Apikey:   "token",
		Endpoint: endpoint,
		Silent:   true,
	}

	listingnoaccess := `{"success":false,"message":"invalid context","code":503}`

	tests := []Unit{
		{
			name:     "download",
			apikey:   "token",
			wantfail: false,
			route:    "/uploads/",
			sendcode: 200,
			sendfile: "../t/t1",
			files:    []string{"cc2c965a"},
			method:   "GET",
			expect:   `cc2c965a successfully downloaded to file t1`,
		},
		{
			name:     "download-catch-empty-response",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads/",
			sendcode: 200,
			files:    []string{"cc2c965a"},
			method:   "GET",
		},
		{
			name:     "download-catch-no-access",
			apikey:   "token",
			wantfail: true,
			route:    "/uploads/",
			sendcode: 503,
			sendjson: listingnoaccess,
			files:    []string{"cc2c965a"},
			method:   "GET",
		},
	}

	for _, unit := range tests {
		var w bytes.Buffer
		unit.route += unit.files[0] + "/file"
		Intercept(unit)
		Check(t, unit, &w, Download(&w, conf, unit.files))

		if unit.sendfile != "" {
			file := filepath.Base(unit.sendfile)
			if _, err := os.Stat(file); err == nil {
				os.Remove(file)
			}
		}
	}
}
