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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"codeberg.org/scip/ephemerup/upctl/cfg"
	"os"
	"strings"
)

var (
	cfgFile string
)

func completion(cmd *cobra.Command, mode string) error {
	switch mode {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return errors.New("Invalid shell parameter! Valid ones: bash|zsh|fish|powershell")
	}
}

func Execute() {
	var (
		conf           cfg.Config
		ShowVersion    bool
		ShowCompletion string
	)

	var rootCmd = &cobra.Command{
		Use:   "upctl [options]",
		Short: "upload api client",
		Long:  `Manage files on an upload api server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ShowVersion {
				fmt.Println(cfg.Getversion())
				return nil
			}

			if len(ShowCompletion) > 0 {
				return completion(cmd, ShowCompletion)
			}

			if len(args) == 0 {
				return errors.New("No command specified!")
			}

			return nil
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(cmd, &conf)
		},
	}

	// options
	rootCmd.PersistentFlags().BoolVarP(&ShowVersion, "version", "v", false, "Print program version")
	rootCmd.PersistentFlags().BoolVarP(&conf.Debug, "debug", "d", false, "Enable debugging")
	rootCmd.PersistentFlags().BoolVarP(&conf.Silent, "silent", "s", false, "Disable progress bar and other noise")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "custom config file")
	rootCmd.PersistentFlags().IntVarP(&conf.Retries, "retries", "r", 3, "How often shall we retry to access our endpoint")
	rootCmd.PersistentFlags().StringVarP(&conf.Endpoint, "endpoint", "p", "http://localhost:8080/api/v1", "upload api endpoint url")
	rootCmd.PersistentFlags().StringVarP(&conf.Apikey, "apikey", "a", "", "Api key to use")

	rootCmd.AddCommand(UploadCommand(&conf))
	rootCmd.AddCommand(ListCommand(&conf))
	rootCmd.AddCommand(DeleteCommand(&conf))
	rootCmd.AddCommand(DescribeCommand(&conf))
	rootCmd.AddCommand(DownloadCommand(&conf))
	rootCmd.AddCommand(ModifyCommand(&conf))

	// forms are being handled with its own subcommand
	rootCmd.AddCommand(FormCommand(&conf))

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// initialize viper, read config and ENV, bind flags
func initConfig(cmd *cobra.Command, cfg *cfg.Config) error {
	v := viper.New()
	viper.SetConfigType("hcl")

	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("upctl")

		// default location[s]
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.config/upctl")
	}

	// ignore read errors, report all others
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// fmt.Println("Using config file:", v.ConfigFileUsed())

	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	v.SetEnvPrefix("upctl")

	// map flags to viper
	return bindFlags(cmd, v)
}

// bind flags to viper settings (env+cfgfile)
func bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	var fail error
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// map flag name to config variable
		configName := f.Name

		// use config variable if flag is not set and config is set
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			if err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
				fail = err
			}
		}
	})

	return fail
}
