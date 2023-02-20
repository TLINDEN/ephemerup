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
	"archive/zip"
	"fmt"
	"github.com/gin-gonic/gin"
	//"github.com/gin-gonic/gin/binding"
	//"encoding/json"
	//"github.com/google/uuid"
	"github.com/tlinden/up/upd/cfg"
	bolt "go.etcd.io/bbolt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	//"strings"
	"time"
)

type Result struct {
	success bool
	url     string
	error   string
}

// Binding from JSON, data coming from user, not tainted
type Meta struct {
	Expire string `form:"expire"`
}

// stores 1 upload object, gets into db
type Upload struct {
	Id       string    `json:"id"`
	Expire   string    `json:"expire"`
	File     string    `json:"file"`    // final filename (visible to the downloader)
	Members  []string  `json:"members"` // contains multiple files, so File is an archive
	Uploaded time.Time `json:"uploaded"`
}

func Log(format string, values ...any) {
	fmt.Fprintf(gin.DefaultWriter, "[GIN] "+format+"\n", values...)
}

func Ts() string {
	t := time.Now()
	return t.Format("2006-01-02-15-04-")
}

func NormalizeFilename(file string) string {
	r := regexp.MustCompile(`[^\w\d\-_\\.]`)

	return Ts() + r.ReplaceAllString(file, "")
}

func Runserver(cfg *cfg.Config, args []string) error {
	dst := cfg.StorageDir
	router := gin.Default()
	router.SetTrustedProxies(nil)
	api := router.Group(cfg.ApiPrefix)

	db, err := bolt.Open(cfg.DbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	{
		api.POST("/putfile", func(c *gin.Context) {
			uri, err := Putfile(c, cfg, db)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"code":    http.StatusOK,
					"message": uri,
					"success": true,
				})
			}
		})

		api.GET("/getfile/:id/:file", func(c *gin.Context) {
			// deliver  a file and delete  it after a delay  (FIXME: check
			// when gin  has done delivering it?). Redirect  to the static
			// handler for actual delivery.
			id := c.Param("id")
			file := c.Param("file")
			c.Request.URL.Path = cfg.ApiPrefix + "/static/" + id + "/" + file
			filename := filepath.Join(dst, id, file)

			if _, err := os.Stat(filename); err == nil {
				go func() {
					time.Sleep(500 * time.Millisecond)
					cleanup(filepath.Join(dst, id))
				}()
			}

			router.HandleContext(c)
		})

		// actual  delivery of static  files, uri's  must be known  to the
		// user, mostly being redirected here internally from /f
		api.Static("/static", dst)
	}

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "welcome to upload api, use /api enpoint!")
	})

	router.Run(cfg.Listen)

	return nil
}

// cleanup an upload directory, either because  we got an error in the
// middle of an upload or something else  went wrong. we fork off a go
// routine because this may block.
func cleanup(dir string) {
	go func() {
		err := os.RemoveAll(dir)
		if err != nil {
			Log("Failed to remove dir %s: %s", dir, err.Error())
		}
	}()
}

func zipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}
