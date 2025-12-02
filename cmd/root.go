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
package cmd

import (
	"errors"
	"fmt"

	"github.com/knadh/koanf/parsers/hcl"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"

	flag "github.com/spf13/pflag"

	"github.com/alecthomas/repr"
	"codeberg.org/scip/ephemerup/api"
	"codeberg.org/scip/ephemerup/cfg"

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	cfgFile string
)

func Execute() error {
	var (
		conf        cfg.Config
		ShowVersion bool
	)

	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	f.BoolVarP(&ShowVersion, "version", "v", false, "Print program version")
	f.StringVarP(&cfgFile, "config", "c", "", "custom config file")
	f.BoolVarP(&conf.Debug, "debug", "d", false, "Enable debugging")
	f.StringVarP(&conf.Listen, "listen", "l", ":8080", "listen to custom ip:port (use [ip]:port for ipv6)")
	f.StringVarP(&conf.StorageDir, "storagedir", "s", "/tmp", "storage directory for uploaded files")
	f.StringVarP(&conf.ApiPrefix, "apiprefix", "a", "", "API endpoint path")
	f.StringVarP(&conf.Url, "url", "u", "", "HTTP endpoint w/o path")
	f.StringVarP(&conf.DbFile, "dbfile", "D", "/tmp/uploads.db", "Bold database file to use")
	f.StringVarP(&conf.Super, "super", "", "", "The API Context which has permissions on all contexts")
	f.StringVarP(&conf.Frontpage, "frontpage", "", "welcome to upload api, use /api enpoint!",
		"Content or filename to be displayed on / in case someone visits")
	f.StringVarP(&conf.Formpage, "formpage", "", "", "Content or filename to be displayed for forms (must be a go template)")

	// server settings
	f.BoolVarP(&conf.V4only, "ipv4", "4", false, "Only listen on ipv4")
	f.BoolVarP(&conf.V6only, "ipv6", "6", false, "Only listen on ipv6")

	f.BoolVarP(&conf.Prefork, "prefork", "p", false, "Prefork server threads")
	f.StringVarP(&conf.AppName, "appname", "n", "ephemerupd "+conf.GetVersion(), "App name to say hi as")
	f.IntVarP(&conf.BodyLimit, "bodylimit", "b", 10250000000, "Max allowed upload size in bytes")

	if err := f.Parse(os.Args[1:]); err != nil {
		return err
	}

	// exclude -6 and -4
	if conf.V4only && conf.V6only {
		return errors.New("You cannot mix -4 and -6!")
	}

	// config provider
	var k = koanf.New(".")

	// Load the config files provided in the commandline or the default locations
	var configfiles []string
	configfile, _ := f.GetString("config")
	if configfile != "" {
		configfiles = []string{configfile}
	} else {
		configfiles = []string{
			"/etc/ephemerup.hcl", "/usr/local/etc/ephemerup.hcl", // unix variants
			filepath.Join(os.Getenv("HOME"), ".config", "ephemerup", "ephemerup.hcl"),
			filepath.Join(os.Getenv("HOME"), ".ephemerup"),
			"ephemerup.hcl",
		}
	}
	repr.Print(configfiles)
	for _, cfgfile := range configfiles {
		if _, err := os.Stat(cfgfile); err == nil {
			if err := k.Load(file.Provider(cfgfile), hcl.Parser(true)); err != nil {
				return errors.New("error loading config file: " + err.Error())
			}
		}
		// else: we ignore the file if it doesn't exists
	}

	// env overrides config file
	if err := k.Load(env.Provider("EPHEMERUPD_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "EPHEMERUPD_")), "_", ".", -1)
	}), nil); err != nil {
		return errors.New("error loading environment: " + err.Error())
	}

	// command line overrides env
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		return errors.New("error loading config: " + err.Error())
	}

	// fetch values
	if err := k.Unmarshal("", &conf); err != nil {
		return errors.New("error unmarshalling: " + err.Error())
	}

	// there may exist some api context variables
	GetApicontextsFromEnv(&conf)

	if conf.Debug {
		repr.Print(conf)
	}

	// Frontpage?
	if conf.Frontpage != "" {
		if _, err := os.Stat(conf.Frontpage); err == nil {
			// it's a filename, try to use it
			content, err := ioutil.ReadFile(conf.Frontpage)
			if err != nil {
				return errors.New("error loading config: " + err.Error())
			}

			// replace the filename
			conf.Frontpage = string(content)
		}
	}

	// Formpage?
	if conf.Formpage != "" {
		if _, err := os.Stat(conf.Formpage); err == nil {
			// it's a filename, try to use it
			content, err := ioutil.ReadFile(conf.Formpage)
			if err != nil {
				return errors.New("error loading config: " + err.Error())
			}

			// replace the filename
			conf.Formpage = string(content)
		}
	} else {
		// use builtin default
		conf.Formpage = formtemplate
	}

	switch {
	case ShowVersion:
		fmt.Println(cfg.Getversion())
		return nil
	default:
		conf.ApplyDefaults()
		return api.Runserver(&conf, flag.Args())
	}
}

/*
   Get a list of Api Contexts from ENV. Useful for use with k8s secrets.

   Multiple env vars are supported in this format:

   EPHEMERUPD_CONTEXT_$(NAME)="<context>:<key>"

eg:

   EPHEMERUPD_CONTEXT_SUPPORT="support:tymag-fycyh-gymof-dysuf-doseb-puxyx"
                      ^^^^^^^- doesn't matter.

   Modifies cfg.Config directly
*/
func GetApicontextsFromEnv(conf *cfg.Config) {
	contexts := []cfg.Apicontext{}

	for _, envvar := range os.Environ() {
		pair := strings.SplitN(envvar, "=", 2)
		if strings.HasPrefix(pair[0], "EPHEMERUPD_CONTEXT_") {
			c := strings.SplitN(pair[1], ":", 2)
			if len(c) == 2 {
				contexts = append(contexts, cfg.Apicontext{Context: c[0], Key: c[1]})
			}
		}
	}

	contexts = append(contexts, conf.Apicontexts...)

	conf.Apicontexts = contexts
}
