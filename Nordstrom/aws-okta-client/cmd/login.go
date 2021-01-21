/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
)

var env string

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Retrieve Okta session credentials required to access AWS roles",
	RunE:  loginRun,
}

func init() {
	RootCmd.AddCommand(loginCmd)
	loginCmd.Flags().BoolVarP(&pci, "pci", "p", false, "Create PCI Okta MFA session - required for accessing PCI AWS accounts (ommit for Non-PCI)")
	loginCmd.Flags().BoolVarP(&dev, "dev", "d", false, "")
	loginCmd.Flags().MarkHidden("dev")
}

func loginRun(cmd *cobra.Command, args []string) error {

	u, err := lib.UpdateHandler(Version)
	if err != nil {
		log.Warn(err)
	}

	resp, od, err := u.Check()
	if err != nil {
		log.Warn(err)
	}

	if od {
		if strings.Contains(resp, "REQUIRED") { //forces exit if out of date and update dialog includes string `REQUIRED`
			log.Errorf("*Required* %s", resp)
		} else {
			log.Warnf("%s", resp)
		}
	}

	if pci {
		env = "pci"
	} else {
		env = "nonpci"
	}

	opts := lib.OktaProviderOptions{}

	clientOpts := lib.ProvidorOptions{}

	if err := lib.DeleteOktaSession(Keyring); err != nil {
		log.Error(err)
		return err
	}

	cfgOps := &lib.ProvidorOptions{
		Stage:       stage,
		Environment: environment,
		LogLevel:    logLevel,
	}

	cfg, err := lib.NewBaseConfig(*cfgOps)
	if err != nil {
		log.Errorf("Unable to set BaseConfig: (%s)", err)
	}

	p, err := lib.NewLoginProvider(Keyring, opts, clientOpts, cfg)
	if err != nil {
		log.Error(err)
		return err
	}

	// Retrieves credentials from Okta, does sign in, etc.
	_, err = p.RetrieveOktaToken()
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
