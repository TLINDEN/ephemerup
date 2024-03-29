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
package common

import (
	"regexp"
	"strconv"
	"time"
)

// https://gist.github.com/rhcarvalho/9338c3ff8850897c68bc74797c5dc25b

// Timestamp is like  time.Time, but knows how to  unmarshal from JSON
// Unix timestamp  numbers or RFC3339  strings, and marshal  back into
// the same JSON representation.
type Timestamp struct {
	time.Time
	rfc3339 bool
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.rfc3339 {
		return t.Time.MarshalJSON()
	}
	return t.formatUnix()
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	err := t.Time.UnmarshalJSON(data)
	if err != nil {
		return t.parseUnix(data)
	}
	t.rfc3339 = true
	return nil
}

func (t Timestamp) formatUnix() ([]byte, error) {
	sec := float64(t.Time.UnixNano()) * float64(time.Nanosecond) / float64(time.Second)
	return strconv.AppendFloat(nil, sec, 'f', -1, 64), nil
}

func (t *Timestamp) parseUnix(data []byte) error {
	f, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	t.Time = time.Unix(0, int64(f*float64(time.Second/time.Nanosecond)))
	return nil
}

/*
   We could use time.ParseDuration(), but this doesn't support days.

   We  could also  use github.com/xhit/go-str2duration/v2,  which does
   the job,  but it's  just another dependency,  just for  this little
   gem. And  we don't need a  time.Time value.

   Convert a  duration into  seconds (int).
   Valid  time units  are "s", "m", "h" and "d".
*/
func Duration2int(duration string) int {
	re := regexp.MustCompile(`(\d+)([dhms])`)
	seconds := 0

	for _, match := range re.FindAllStringSubmatch(duration, -1) {
		if len(match) == 3 {
			v, _ := strconv.Atoi(match[1])
			switch match[2][0] {
			case 'd':
				seconds += v * 86400
			case 'h':
				seconds += v * 3600
			case 'm':
				seconds += v * 60
			case 's':
				seconds += v
			}
		}
	}

	return seconds
}
