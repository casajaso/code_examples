/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:                   "version",
	Short:                 fmt.Sprintf("Print version information"),
	Run:                   versionRun,
	DisableFlagsInUseLine: true,
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

// returns version info
func versionRun(cmd *cobra.Command, args []string) {
	fmt.Printf("%s Release: (%s)\n", os.Args[0], Version)
	return
}
