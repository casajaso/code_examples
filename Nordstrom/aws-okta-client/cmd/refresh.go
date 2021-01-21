/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
)

// refreshCmd represents the refresh command
var refreshCmd = &cobra.Command{
	Use:     "refresh -p <profile-name>",
	Short:   "Refresh Okta/AWS session token experation",
	Hidden:  false,
	PreRunE: refreshPre,
	RunE:    refreshRun,
}

func init() {
	RootCmd.AddCommand(refreshCmd)
	refreshCmd.Flags().StringVarP(&ProfileName, "profile-name", "p", "nordstrom-federated", "Set profile name")
	refreshCmd.Flags().DurationVarP(&assumeRoleTTL, "assume-role-ttl", "t", 0, "Set AWS Session duration. Options [NonPCI: 0h15m0s - 1h0m0s, PCI: 0h15m0s]")
}

func refreshPre(cmd *cobra.Command, args []string) error {

	// determine if --profile flag is set by user
	if cmd.Flags().Changed("profile-name") {
		profileFlagSet = true
	}

	// determine if --assume-role-ttl flag is set by user
	if cmd.Flags().Changed("assume-role-ttl") {
		assumeRoleTTLFlagSet = true
	}

	if err := loadStringFlagFromEnv(cmd, "profile-name", "AWS_OKTA_DEFAULT_PROFILE", &ProfileName, &profileFlagSet); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to parse profile from AWS_OKTA_DEFAULT_PROFILE")
	}

	if profileFlagSet {
		ProfileName = ProfileName
	} else if len(args) > 0 {
		ProfileName = args[0]
	} else {
		ProfileName = ProfileName
	}

	return nil
}

func refreshRun(cmd *cobra.Command, args []string) error {

	cfgFile, err := lib.NewProfileProvider("config")
	if err != nil {
		log.Error(err)
		return err
	}

	prf, err := cfgFile.Parse(ProfileName, true)
	if err != nil {
		log.Error(err)
		return err
	}

	keys, err := Keyring.Keys()
	if err != nil {
		log.Error(err)
		return err
	}

	var OktaSessionKey string
	for _, k := range keys {
		if strings.Contains(k, lib.OktaSessionName) {
			OktaSessionKey = k
		}
	}

	if len(OktaSessionKey) < 20 {
		err := fmt.Errorf("key not found in keyring")
		log.Errorf("Unable to retrieve %s token: (%s)", lib.OktaSessionName, err)
		return err
	}

	var item keyring.Item
	item, err = Keyring.Get(OktaSessionKey)
	if err != nil {
		log.Error(err)
		return err
	}

	token := &lib.OktaSessionToken{}

	if err = json.Unmarshal(item.Data, token); err != nil {
		log.Error(err)
		return err
	}

	stage := token.Stage
	environment := token.Environment

	cfgOps := &lib.ProvidorOptions{
		Stage:       stage,
		Environment: environment,
		LogLevel:    logLevel,
	}

	cfg, err := lib.NewBaseConfig(*cfgOps)
	if err != nil {
		log.Errorf("Unable to set BaseConfig: (%s)", err)
	}

	fmt.Fprintf(os.Stdout, "Refreshing Okta Session credentials\n")

	err = lib.RefreshOktaSessionToken(Keyring, item, cfg)
	if err != nil {
		log.Errorf("Unable to refresh Credentials: (%s)", err)
	}

	item, err = Keyring.Get(OktaSessionKey)
	if err != nil {
		log.Error(err)
		return err
	}

	if err = json.Unmarshal(item.Data, &token); err != nil {
		log.Error(err)
		return err
	}

	var assumeRoleDuration time.Duration

	if assumeRoleTTL > 0 {
		assumeRoleDuration = assumeRoleTTL
	} else {
		assumeRoleDuration = cfg.AWS.SessionDefault
	}

	awsOps := &lib.ProvidorOptions{
		Keyring:            Keyring,
		ProfileName:        ProfileName,
		AssumeRoleDuration: assumeRoleDuration,
		Config:             cfg,
	}

	p, err := lib.AWSCredsProvider(*awsOps)
	if err != nil {
		log.Error(err)
		return err
	}

	fmt.Fprintf(os.Stdout, "Refreshing AWS Session credentials\n")

	p.DeleteAWSSession()

	_, _, err = p.Retrieve(prf)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
