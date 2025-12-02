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
	"github.com/spf13/cobra"
	"codeberg.org/scip/ephemerup/common"
	"codeberg.org/scip/ephemerup/upctl/cfg"
	"codeberg.org/scip/ephemerup/upctl/lib"
	"os"
)

func UploadCommand(conf *cfg.Config) *cobra.Command {
	var uploadCmd = &cobra.Command{
		Use:   "upload [options] [file ..]",
		Short: "Upload files",
		Long:  `Upload files to an upload api.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("No files specified to upload!")
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.UploadFiles(os.Stdout, conf, args)
		},
	}

	// options
	uploadCmd.PersistentFlags().StringVarP(&conf.Expire, "expire", "e", "", "Expire setting: asap or duration (accepted shortcuts: dmh)")
	uploadCmd.PersistentFlags().StringVarP(&conf.Description, "description", "D", "", "Description of the form")

	uploadCmd.Aliases = append(uploadCmd.Aliases, "up")
	uploadCmd.Aliases = append(uploadCmd.Aliases, "u")

	return uploadCmd
}

func ListCommand(conf *cfg.Config) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list [options] [file ..]",
		Short: "List uploads",
		Long:  `List uploads`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.List(os.Stdout, conf, args, common.TypeUpload)
		},
	}

	// options
	listCmd.PersistentFlags().StringVarP(&conf.Apicontext, "apicontext", "", "", "Filter by given API context")
	listCmd.PersistentFlags().StringVarP(&conf.Query, "query", "q", "", "Filter by given query regexp")

	listCmd.Aliases = append(listCmd.Aliases, "ls")
	listCmd.Aliases = append(listCmd.Aliases, "l")

	return listCmd
}

func DeleteCommand(conf *cfg.Config) *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use:   "delete [options] <id>",
		Short: "Delete an upload",
		Long:  `Delete an upload identified by its id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("No id specified to delete!")
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.Delete(os.Stdout, conf, args, common.TypeUpload)
		},
	}

	deleteCmd.Aliases = append(deleteCmd.Aliases, "rm")
	deleteCmd.Aliases = append(deleteCmd.Aliases, "d")

	return deleteCmd
}

func DescribeCommand(conf *cfg.Config) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "describe [options] upload-id",
		Long:  "Show detailed informations about an upload object.",
		Short: `Describe an upload`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("No id specified to delete!")
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.Describe(os.Stdout, conf, args, common.TypeUpload)
		},
	}

	listCmd.Aliases = append(listCmd.Aliases, "des")
	listCmd.Aliases = append(listCmd.Aliases, "info")
	listCmd.Aliases = append(listCmd.Aliases, "i")

	return listCmd
}

func DownloadCommand(conf *cfg.Config) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "download [options] upload-id",
		Long:  "Download the file associated with an upload object.",
		Short: `Download a file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("No id specified to delete!")
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.Download(os.Stdout, conf, args)
		},
	}

	listCmd.Aliases = append(listCmd.Aliases, "down")
	listCmd.Aliases = append(listCmd.Aliases, "get")
	listCmd.Aliases = append(listCmd.Aliases, "g")
	listCmd.Aliases = append(listCmd.Aliases, "fetch")

	return listCmd
}

func ModifyCommand(conf *cfg.Config) *cobra.Command {
	var uploadModifyCmd = &cobra.Command{
		Use:   "modify [options] <id>",
		Short: "Modify an upload",
		Long:  `Modify an existing upload.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.Modify(os.Stdout, conf, args, common.TypeUpload)
		},
	}

	// options
	uploadModifyCmd.PersistentFlags().StringVarP(&conf.Expire, "expire", "e", "",
		"Expire setting: asap or duration (accepted shortcuts: dmh)")
	uploadModifyCmd.PersistentFlags().StringVarP(&conf.Description, "description", "D", "",
		"Description of the upload")

	uploadModifyCmd.Aliases = append(uploadModifyCmd.Aliases, "mod")
	uploadModifyCmd.Aliases = append(uploadModifyCmd.Aliases, "change")

	return uploadModifyCmd
}
