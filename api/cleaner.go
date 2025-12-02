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
	//"github.com/alecthomas/repr"
	"encoding/json"
	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"time"
)

func DeleteExpiredUploads(conf *cfg.Config, db *Db) error {
	err := db.bolt.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(Bucket))

		if bucket == nil {
			return nil // nothin to delete so far
		}

		err := bucket.ForEach(func(id, j []byte) error {
			upload := &common.Upload{}
			if err := json.Unmarshal(j, &upload); err != nil {
				return fmt.Errorf("unable to unmarshal json: %s", err)
			}

			if IsExpired(conf, upload.Created.Time, upload.Expire) {
				if err := bucket.Delete([]byte(id)); err != nil {
					return nil
				}

				cleanup(filepath.Join(conf.StorageDir, upload.Id))

				Log("Cleaned up upload " + upload.Id)
			}

			return nil
		})

		return err
	})

	return err
}

func BackgroundCleaner(conf *cfg.Config, db *Db) chan bool {
	ticker := time.NewTicker(conf.CleanInterval)
	fmt.Println(conf.CleanInterval)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := DeleteExpiredUploads(conf, db); err != nil {
					Log("Failed to delete eypired uploads: %s", err.Error())
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return done
}
