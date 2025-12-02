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
	"net/smtp"
)

var mailtpl string = `To: %s\r
From: %s\r
Subject: %s\r
\r
%s\r
`

/*
   Send  an  email  via  an   external  mail  gateway.  SMTP  Auth  is
   required. Errors may occur with  a time delay, like server timeouts
   etc. So only call it detached via go routine.
*/
func Sendmail(c *cfg.Config, recipient string, body string, subject string) error {
	// Message.
	message := []byte(fmt.Sprintf(mailtpl, recipient, c.Mail.From, subject, body))

	// Authentication.
	auth := smtp.PlainAuth("", c.Mail.From, c.Mail.Password, c.Mail.Server)

	// Sending email.
	Log("Trying to send mail to %s via %s:%s with subject %s",
		recipient, c.Mail.Server, c.Mail.Port, subject)
	err := smtp.SendMail(c.Mail.Server+":"+c.Mail.Port, auth, c.Mail.From, []string{recipient}, []byte(message))
	if err != nil {
		return err
	}

	return nil
}
