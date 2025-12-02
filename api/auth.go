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
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/keyauth/v2"
	"codeberg.org/scip/ephemerup/cfg"
	"codeberg.org/scip/ephemerup/common"
)

// these vars can be savely global, since they don't change ever
var (
	errMissing = &fiber.Error{
		Code:    403000,
		Message: "Missing API key",
	}

	errInvalid = &fiber.Error{
		Code:    403001,
		Message: "Invalid API key",
	}

	Apikeys []cfg.Apicontext
)

// fill from server: accepted keys
func AuthSetApikeys(keys []cfg.Apicontext) {
	Apikeys = keys
}

// make sure we always return JSON encoded errors
func AuthErrHandler(ctx *fiber.Ctx, err error) error {
	ctx.Status(fiber.StatusForbidden)

	if err == errMissing {
		return ctx.JSON(errMissing)
	}

	return ctx.JSON(errInvalid)
}

// validator hook, validates  incoming api key against  form id, which
// also acts as onetime api key
func AuthValidateOnetimeKey(c *fiber.Ctx, key string, db *Db) (bool, error) {
	resp, err := db.Get("", key, common.TypeForm)
	if err != nil {
		return false, errors.New("Onetime key doesn't match any form id!")
	}

	if len(resp.Forms) != 1 {
		return false, errors.New("db.Get(form) returned no results and no errors!")
	}

	sess, err := Sessionstore.Get(c)
	if err != nil {
		return false, errors.New("Could not retrieve session from Sessionstore: " + err.Error())
	}

	// store the  result into the session, the 'formid'  key tells the
	// upload handler that the apicontext it sees is in fact a form id
	// and has to be deleted if set to asap.
	sess.Set("apicontext", resp.Forms[0].Context)
	sess.Set("formid", key)

	if err := sess.Save(); err != nil {
		return false, errors.New("Unable to save session store!")
	}

	return true, nil
}

// validator hook, called by fiber via server keyauth.New()
func AuthValidateAPIKey(c *fiber.Ctx, key string) (bool, error) {
	// create a new session, it will be thrown away if something fails
	sess, err := Sessionstore.Get(c)
	if err != nil {
		return false, errors.New("Unable to initialize session store from context!")
	}

	// if Apikeys is empty, the server works unauthenticated
	// FIXME: maybe always reject?
	if len(Apikeys) == 0 {
		sess.Set("apicontext", "default")

		if err := sess.Save(); err != nil {
			return false, errors.New("Unable to save session store!")
		}

		return true, nil
	}

	// actual key comparison
	for _, apicontext := range Apikeys {
		hashedAPIKey := sha256.Sum256([]byte(apicontext.Key))
		hashedKey := sha256.Sum256([]byte(key))

		if subtle.ConstantTimeCompare(hashedAPIKey[:], hashedKey[:]) == 1 {
			// apikey matches, register apicontext for later use by the handlers
			sess.Set("apicontext", apicontext.Context)

			if err := sess.Save(); err != nil {
				return false, errors.New("Unable to save session store!")
			}

			return true, nil
		}
	}

	return false, keyauth.ErrMissingOrMalformedAPIKey
}
