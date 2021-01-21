/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:    "logout",
	Short:  "Remove Okta/AWS session tokens from the keychain",
	Hidden: false,
	PreRun: logoutPre,
	RunE:   logoutRun,
}

func init() {
	RootCmd.AddCommand(logoutCmd)
}

func logoutPre(cmd *cobra.Command, args []string) {
}

func logoutRun(cmd *cobra.Command, args []string) error {

	if err := lib.DeleteOktaSession(Keyring); err != nil {
		log.Error(err)
		return err
	}

	if err := lib.DeleteAWSSessions(Keyring); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
