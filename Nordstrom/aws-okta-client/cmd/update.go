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

// updatCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Download and/or install the latest update",
	RunE:    updateRun,
	PreRunE: updatePre,
}

func init() {
	RootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVarP(&install, "install", "i", false, "Install the latest update (not supported on Windows hosts)")
	updateCmd.Flags().BoolVarP(&download, "download", "d", false, "Download the latest update")
}

func updatePre(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("install") {
		installFlagSet = true
	}

	if cmd.Flags().Changed("download") {
		downloadFlagSet = true
	}
	return nil
}

func updateRun(cmd *cobra.Command, args []string) error {

	if installFlagSet {
		if lib.Platform == "windows" {
			log.Errorf("In-place update is not supported on windows hosts. use `-d` or `download` to manually install update")
		}

		u, err := lib.UpdateHandler(Version)
		if err != nil {
			log.Error(err)
			return err
		}

		err = u.Get()
		if err != nil {
			log.Errorf("Failed to download update: (%v)", err)
			return err
		}

		err = u.Install()
		if err != nil {
			log.Errorf("Failed to install update: (%v)", err)
			return err
		}

	} else if downloadFlagSet {
		u, err := lib.UpdateHandler(Version)
		if err != nil {
			log.Error(err)
			return err
		}

		err = u.Get()
		if err != nil {
			log.Errorf("Failed to download update: (%v)", err)
			return err
		}

		err = u.Save()
		if err != nil {
			log.Errorf("Failed to download update: (%v)", err)
			return err
		}

	} else {
		log.Errorf("%s: (`update` requires `-i` (--install) or `-d` (--download))",
			ErrTooFewArguments,
		)
	}

	return nil
}
