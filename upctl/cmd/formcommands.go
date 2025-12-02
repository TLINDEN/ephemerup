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
	//"errors"
	"errors"
	"github.com/spf13/cobra"
	"codeberg.org/scip/ephemerup/common"
	"codeberg.org/scip/ephemerup/upctl/cfg"
	"codeberg.org/scip/ephemerup/upctl/lib"
	"os"
)

func FormCommand(conf *cfg.Config) *cobra.Command {
	var formCmd = &cobra.Command{
		Use:   "form {create|delete|modify|list}",
		Short: "Form commands",
		Long:  `Manage upload forms.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// errors at this stage do not cause the usage to be shown
			//cmd.SilenceUsage = true
			if len(args) == 0 {
				return cmd.Help()
			}
			return nil
		},
	}

	formCmd.Aliases = append(formCmd.Aliases, "frm")
	formCmd.Aliases = append(formCmd.Aliases, "f")

	formCmd.AddCommand(FormCreateCommand(conf))
	formCmd.AddCommand(FormListCommand(conf))
	formCmd.AddCommand(FormDeleteCommand(conf))
	formCmd.AddCommand(FormDescribeCommand(conf))
	formCmd.AddCommand(FormModifyCommand(conf))

	return formCmd
}

func FormCreateCommand(conf *cfg.Config) *cobra.Command {
	var formCreateCmd = &cobra.Command{
		Use:   "create [options]",
		Short: "Create a new form",
		Long:  `Create a new form for consumers so they can upload something.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.CreateForm(os.Stdout, conf)
		},
	}

	// options
	formCreateCmd.PersistentFlags().StringVarP(&conf.Expire, "expire", "e", "",
		"Expire setting: asap or duration (accepted shortcuts: dmh)")
	formCreateCmd.PersistentFlags().StringVarP(&conf.Description, "description", "D", "",
		"Description of the form")
	formCreateCmd.PersistentFlags().StringVarP(&conf.Notify, "notify", "n", "",
		"Email address to get notified when consumer has uploaded files")

	formCreateCmd.Aliases = append(formCreateCmd.Aliases, "add")
	formCreateCmd.Aliases = append(formCreateCmd.Aliases, "+")

	return formCreateCmd
}

func FormModifyCommand(conf *cfg.Config) *cobra.Command {
	var formModifyCmd = &cobra.Command{
		Use:   "modify [options] <id>",
		Short: "Modify a form",
		Long:  `Modify an existing form.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.Modify(os.Stdout, conf, args, common.TypeForm)
		},
	}

	// options
	formModifyCmd.PersistentFlags().StringVarP(&conf.Expire, "expire", "e", "",
		"Expire setting: asap or duration (accepted shortcuts: dmh)")
	formModifyCmd.PersistentFlags().StringVarP(&conf.Description, "description", "D", "",
		"Description of the form")
	formModifyCmd.PersistentFlags().StringVarP(&conf.Notify, "notify", "n", "",
		"Email address to get notified when consumer has uploaded files")

	formModifyCmd.Aliases = append(formModifyCmd.Aliases, "mod")
	formModifyCmd.Aliases = append(formModifyCmd.Aliases, "change")

	return formModifyCmd
}

func FormListCommand(conf *cfg.Config) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list [options]",
		Short: "List formss",
		Long:  `List formss.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.List(os.Stdout, conf, nil, common.TypeForm)
		},
	}

	// options
	listCmd.PersistentFlags().StringVarP(&conf.Apicontext, "apicontext", "", "", "Filter by given API context")
	listCmd.PersistentFlags().StringVarP(&conf.Query, "query", "q", "", "Filter by given query regexp")

	listCmd.Aliases = append(listCmd.Aliases, "ls")
	listCmd.Aliases = append(listCmd.Aliases, "l")

	return listCmd
}

func FormDeleteCommand(conf *cfg.Config) *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use:   "delete [options] <id>",
		Short: "Delete an form",
		Long:  `Delete an form identified by its id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("No id specified to delete!")
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.Delete(os.Stdout, conf, args, common.TypeForm)
		},
	}

	deleteCmd.Aliases = append(deleteCmd.Aliases, "rm")
	deleteCmd.Aliases = append(deleteCmd.Aliases, "d")

	return deleteCmd
}

func FormDescribeCommand(conf *cfg.Config) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "describe [options] form-id",
		Long:  "Show detailed informations about an form object.",
		Short: `Describe an form.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("No id specified to delete!")
			}

			// errors at this stage do not cause the usage to be shown
			cmd.SilenceUsage = true

			return lib.Describe(os.Stdout, conf, args, common.TypeForm)
		},
	}

	listCmd.Aliases = append(listCmd.Aliases, "des")
	listCmd.Aliases = append(listCmd.Aliases, "info")
	listCmd.Aliases = append(listCmd.Aliases, "i")

	return listCmd
}
