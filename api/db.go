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
	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"
	//"github.com/alecthomas/repr"
	bolt "go.etcd.io/bbolt"
	"regexp"
)

const Bucket string = "data"

// wrapper for bolt db
type Db struct {
	bolt *bolt.DB
	cfg  *cfg.Config
}

func NewDb(c *cfg.Config) (*Db, error) {
	b, err := bolt.Open(c.DbFile, 0600, nil)
	db := Db{bolt: b, cfg: c}
	return &db, err
}

func (db *Db) Close() {
	db.bolt.Close()
}

func (db *Db) Insert(id string, entry common.Dbentry) error {
	err := db.bolt.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(Bucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		jsonentry, err := entry.Marshal()
		if err != nil {
			return fmt.Errorf("json marshalling failure: %s", err)
		}

		err = bucket.Put([]byte(id), []byte(jsonentry))
		if err != nil {
			return fmt.Errorf("insert data: %s", err)
		}

		return nil
	})
	if err != nil {
		Log("DB error: %s", err.Error())
	}

	return err
}

func (db *Db) Delete(apicontext string, id string) error {
	err := db.bolt.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(Bucket))

		if bucket == nil {
			return fmt.Errorf("id %s not found", id)
		}

		j := bucket.Get([]byte(id))

		if len(j) == 0 {
			return fmt.Errorf("id %s not found", id)
		}

		entryContext, err := common.GetContext(j)
		if err != nil {
			return fmt.Errorf("unable to unmarshal json: %s", err)
		}

		if (apicontext != "" && (db.cfg.Super == apicontext || entryContext == apicontext)) || apicontext == "" {
			return bucket.Delete([]byte(id))
		}

		return nil
	})

	if err != nil {
		Log("DB error: %s", err.Error())
	}

	return err
}

func (db *Db) List(apicontext string, filter string, query string, t int) (*common.Response, error) {
	response := &common.Response{}
	qr := regexp.MustCompile(query)

	err := db.bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(Bucket))
		if bucket == nil {
			return nil
		}

		err := bucket.ForEach(func(id, j []byte) error {
			allowed := false
			entry, err := common.Unmarshal(j, t)

			if err != nil {
				return fmt.Errorf("unable to unmarshal json: %s", err)
			}

			if !entry.IsType(t) {
				return nil
			}

			var entryContext string
			if t == common.TypeUpload {
				entryContext = entry.(*common.Upload).Context
			} else {
				entryContext = entry.(*common.Form).Context
			}

			// check if the user is allowed to list this entry
			if apicontext != "" && db.cfg.Super != apicontext {
				// authenticated user but not member of super
				// only return the uploads matching her context
				if apicontext == entryContext {
					// unless a filter OR no filter specified
					if (filter != "" && entryContext == filter) || filter == "" {
						allowed = true
					}
				}
			} else {
				// return all, because we operate a public service or current==super
				if (filter != "" && entryContext == filter) || filter == "" {
					allowed = true
				}
			}

			if allowed {
				// user is allowed to view this entry, check if she also wants to see it
				if query != "" {
					if entry.MatchDescription(qr) ||
						entry.MatchExpire(qr) ||
						entry.MatchCreated(qr) ||
						entry.MatchFile(qr) {
						allowed = true
					} else {
						allowed = false
					}
				}
			}

			if allowed {
				// ok, legit and wanted
				response.Append(entry)
			}

			return nil
		})

		return err // might be nil as well
	})

	return response, err
}

// we only return one obj here, but could return more later
// FIXME: turn the id into a filter and call (Uploads|Forms)List(), same code!
func (db *Db) Get(apicontext string, id string, t int) (*common.Response, error) {
	response := &common.Response{}

	err := db.bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(Bucket))
		if bucket == nil {
			return nil
		}

		j := bucket.Get([]byte(id))
		if j == nil {
			return fmt.Errorf("No upload object found with id %s", id)
		}

		entry, err := common.Unmarshal(j, t)
		if err != nil {
			return fmt.Errorf("unable to unmarshal json: %s", err)
		}

		var entryContext string
		if t == common.TypeUpload {
			entryContext = entry.(*common.Upload).Context
		} else {
			entryContext = entry.(*common.Form).Context
		}

		if (apicontext != "" && (db.cfg.Super == apicontext || entryContext == apicontext)) || apicontext == "" {
			// allowed if no context (public or download)
			// or if context matches or if context==super
			response.Append(entry)
		}

		return nil
	})

	return response, err
}

// a wrapper around Lookup() which extracts the 1st upload, if any
func (db *Db) Lookup(apicontext string, id string, t int) (*common.Response, error) {
	response, err := db.Get(apicontext, id, t)

	if err != nil {
		// non existent db entry with that id, or other db error, see logs
		return &common.Response{}, fmt.Errorf("No upload object found with id %s", id)
	}

	if len(response.Uploads) == 0 {
		return &common.Response{}, fmt.Errorf("No upload object found with id %s", id)
	}

	return response, nil
}
