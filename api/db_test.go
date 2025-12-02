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
	"os"
	"testing"
	"time"

	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"
	"github.com/maxatome/go-testdeep/td"
)

func finalize(db *Db) {
	if db.bolt != nil {
		db.Close()
	}
	if _, err := os.Stat(db.cfg.DbFile); err == nil {
		os.Remove(db.cfg.DbFile)
	}
}

func TestNew(t *testing.T) {
	var tests = []struct {
		name     string
		dbfile   string
		wantfail bool
	}{
		{"opennew", "test.db", false},
		{"openfail", "/hopefully/not/existing/directory/test.db", true},
	}

	for _, tt := range tests {
		c := &cfg.Config{DbFile: tt.dbfile}
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDb(c)
			defer finalize(db)
			if err != nil && !tt.wantfail {
				t.Error("expected: &Db{}, got err: " + err.Error())
			}

			if err == nil && tt.wantfail {
				t.Error("expected: fail, got &Db{}")
			}
		})
	}
}

const timeformat string = "2006-01-02T15:04:05.000Z"

var dbtests = []struct {
	name     string
	dbfile   string
	wantfail bool
	id       string
	context  string
	ts       string
	filter   string
	query    string
	upload   common.Upload
	form     common.Form
}{
	{
		"upload", "test.db", false, "1", "foo",
		"2023-03-10T11:45:00.000Z", "", "",
		common.Upload{
			Id: "1", Expire: "asap", File: "none", Context: "foo",
			Created: common.Timestamp{}, Type: common.TypeUpload},
		common.Form{},
	},
	{
		"form", "test.db", false, "2", "foo",
		"2023-03-10T11:45:00.000Z", "", "",
		common.Upload{},
		common.Form{
			Id: "1", Expire: "asap", Description: "none", Context: "foo",
			Created: common.Timestamp{}, Type: common.TypeForm},
	},
}

/*
We  need to  test the  whole Db  operation in  one run,  because it
doesn't work well if using a global Db.
*/
func TestDboperation(t *testing.T) {
	for _, tt := range dbtests {
		c := &cfg.Config{DbFile: tt.dbfile}
		t.Run(tt.name, func(t *testing.T) {
			// create new bbolt db
			db, err := NewDb(c)
			defer finalize(db)

			if err != nil {
				t.Error("Could not open new DB: " + err.Error())
			}

			if tt.upload.Id != "" {
				// set ts
				ts, err := time.Parse(timeformat, tt.ts)
				if err != nil {
					t.Error("Could not parse time: " + err.Error())
				}

				tt.upload.Created = common.Timestamp{Time: ts}

				// create new upload db object
				err = db.Insert(tt.id, tt.upload)
				if err != nil {
					t.Error("Could not insert new upload object: " + err.Error())
				}

				// fetch it
				response, err := db.Get(tt.context, tt.id, common.TypeUpload)
				if err != nil {
					t.Error("Could not fetch upload object: " + err.Error())
				}

				// is it there?
				if len(response.Uploads) != 1 {
					t.Error("db.Get() did not return an upload obj")
				}

				// compare times
				if !tt.upload.Created.Time.Equal(response.Uploads[0].Created.Time) {
					t.Errorf("Timestamps don't match!\ngot: %s\nexp: %s\n",
						response.Uploads[0].Created, tt.upload.Created)
				}

				// equal them artificially,  because otherwise td will
				// fail because of time.Time.wall+ext, or TZ is missing
				response.Uploads[0].Created = tt.upload.Created

				// compare
				td.Cmp(t, response.Uploads[0], &tt.upload, tt.name)

				// fetch list
				response, err = db.List(tt.context, tt.filter, tt.query, common.TypeUpload)
				if err != nil {
					t.Error("Could not fetch uploads list: " + err.Error())
				}

				// is it there?
				if len(response.Uploads) != 1 {
					t.Error("db.List() did not return upload obj[s]")
				}

				// delete
				err = db.Delete(tt.context, tt.id)
				if err != nil {
					t.Error("Could not delete upload obj: " + err.Error())
				}

				// fetch again, shall return empty
				_, err = db.Get(tt.context, tt.id, common.TypeUpload)
				if err == nil {
					t.Error("Could fetch upload object again although we deleted it")
				}
			}

			if tt.form.Id != "" {
				// set ts
				ts, err := time.Parse(timeformat, tt.ts)
				if err != nil {
					t.Error("Could not parse time: " + err.Error())
				}
				tt.form.Created = common.Timestamp{Time: ts}

				// create new form db object
				err = db.Insert(tt.id, tt.form)
				if err != nil {
					t.Error("Could not insert new form object: " + err.Error())
				}

				// fetch it
				response, err := db.Get(tt.context, tt.id, common.TypeForm)
				if err != nil {
					t.Error("Could not fetch form object: " + err.Error())
				}

				// is it there?
				if len(response.Forms) != 1 {
					t.Error("db.Get() did not return an form obj")
				}

				// compare times
				if !tt.form.Created.Time.Equal(response.Forms[0].Created.Time) {
					t.Errorf("Timestamps don't match!\ngot: %s\nexp: %s\n",
						response.Forms[0].Created, tt.form.Created)
				}

				// equal them artificially,  because otherwise td will
				// fail because of time.Time.wall+ext, or TZ is missing
				response.Forms[0].Created = tt.form.Created

				// compare
				td.Cmp(t, response.Forms[0], &tt.form, tt.name)

				// fetch list
				response, err = db.List(tt.context, tt.filter, tt.query, common.TypeForm)
				if err != nil {
					t.Error("Could not fetch forms list: " + err.Error())
				}

				// is it there?
				if len(response.Forms) != 1 {
					t.Error("db.FormsList() did not return form obj[s]")
				}

				// delete
				err = db.Delete(tt.context, tt.id)
				if err != nil {
					t.Error("Could not delete form obj: " + err.Error())
				}

				// fetch again, shall return empty
				_, err = db.Get(tt.context, tt.id, common.TypeForm)
				if err == nil {
					t.Error("Could fetch form object again although we deleted it")
				}
			}
		})
	}
}
