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
	"archive/zip"
	"errors"
	"github.com/gofiber/fiber/v2"
	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// cleanup an upload directory, either because  we got an error in the
// middle of an upload or something else  went wrong.
func cleanup(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		Log("Failed to remove dir %s: %s", dir, err.Error())
	}
}

// Extract form file[s] and store them on disk, returns a list of files
func SaveFormFiles(c *fiber.Ctx, cfg *cfg.Config, files []*multipart.FileHeader, id string) ([]string, error) {
	members := []string{}
	for _, file := range files {
		filename, _ := common.Untaint(filepath.Base(file.Filename), cfg.RegNormalizedFilename)
		path := filepath.Join(cfg.StorageDir, id, filename)
		members = append(members, filename)
		Log("Received: %s => %s/%s", file.Filename, id, filename)

		if err := c.SaveFile(file, path); err != nil {
			cleanup(filepath.Join(cfg.StorageDir, id))
			return nil, err
		}
	}

	return members, nil
}

// generate return url. in case of multiple files, zip and remove them
func ProcessFormFiles(cfg *cfg.Config, members []string, id string) (string, string, error) {
	returnUrl := ""
	Filename := ""

	if len(members) == 1 {
		returnUrl = strings.Join([]string{cfg.Url, "download", id, members[0]}, "/")
		Filename = members[0]
	} else {
		zipfile := Ts() + "data.zip"
		tmpzip := filepath.Join(cfg.StorageDir, zipfile)
		finalzip := filepath.Join(cfg.StorageDir, id, zipfile)
		iddir := filepath.Join(cfg.StorageDir, id)

		if err := ZipDir(iddir, tmpzip); err != nil {
			cleanup(iddir)
			Log("zip error")
			return "", "", err
		}

		if err := os.Rename(tmpzip, finalzip); err != nil {
			cleanup(iddir)
			return "", "", err
		}

		returnUrl = strings.Join([]string{cfg.Url, "download", id, zipfile}, "/")
		Filename = zipfile

		// clean up after us
		go func() {
			for _, file := range members {
				if err := os.Remove(filepath.Join(cfg.StorageDir, id, file)); err != nil {
					Log("ERROR: unable to delete %s: %s", file, err)
				}
			}
		}()
	}

	return returnUrl, Filename, nil
}

// Create a zip archive from a directory
// FIXME: -e option, if any, goes here
func ZipDir(directory, zipfilename string) error {
	f, err := os.Create(zipfilename)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	failure := make(chan string)

	// don't chdir the server itself
	go func() {
		defer wg.Done()

		if err := os.Chdir(directory); err != nil {
			failure <- "Failed to changedir: " + err.Error()
			return
		}
		newDir, err := os.Getwd()
		if err != nil {
			failure <- "Failed to get cwd: " + err.Error()
		}

		if newDir != directory {
			failure <- "Failed to changedir!"
		}

		err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			// 2. Go through all the files of the directory
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
			header.Name = path
			//Log("a: <%s>, b: <%s>, rel: <%s>", filepath.Dir(directory), path, header.Name)
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

		if err != nil {
			failure <- "Failed to zip directory: " + err.Error()
		} else {
			close(failure)
		}
	}()

	wg.Wait()

	goterr := <-failure

	if goterr != "" {
		return errors.New(goterr)
	}

	return err
}
