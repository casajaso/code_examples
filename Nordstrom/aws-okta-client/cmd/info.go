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

	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
)

// refreshCmd represents the refresh command
var infoCmd = &cobra.Command{
	Use:     "info -p <profile-name>",
	Short:   "Display Okta/AWS session token experation",
	Hidden:  false,
	PreRunE: infoPre,
	RunE:    infoRun,
}

var (
	isPCI bool
	e     string
)

func init() {
	RootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringVarP(&ProfileName, "profile-name", "p", "nordstrom-federated", "Set profile name")
	infoCmd.Flags().DurationVarP(&assumeRoleTTL, "assume-role-ttl", "t", 0, "Set AWS Session duration. Options [NonPCI: 0h15m0s - 1h0m0s, PCI: 0h15m0s]")
	infoCmd.Flags().MarkHidden("assume-role-ttl")
}

func infoPre(cmd *cobra.Command, args []string) error {

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

func infoRun(cmd *cobra.Command, args []string) error {

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

	item, err := Keyring.Get(OktaSessionKey)
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

	ra, err := prf.GetKey("_role_arn")
	if err != nil {
		log.Error(err)
		return err
	}

	creds, expire, err := p.GetTokenInfo()
	if err != nil {
		log.Error(err)
	}

	envs := []string{
		"Prod",
		"NonProd",
		"PCIProd",
		"PCINonProd",
	}

	_env := strings.Split(strings.Split(ra.String(), ":")[5][5:], "_")[0]

	for _, _e := range envs {
		if _env == _e {
			e = _e
		}
	}

	if e == "" {
		e = "undefined"
	}

	fmt.Fprintf(os.Stdout, "Okta Session:\n\tDataClassification: %v\n\tExpiration: %s (%s)\n",
		token.Environment,
		token.SessionToken.Expiry.UTC(),
		time.Until(token.SessionToken.Expiry),
	)

	fmt.Fprintf(os.Stdout, "AWS Session:\n\tAccountId: %s\n\tRoleName: %s\n\tEnvironment: %s\n\tExpiration: %s (%s)\n",
		strings.Split(ra.String(), ":")[4],
		strings.Split(ra.String(), ":")[5][5:],
		e,
		creds.Expiration.UTC(),
		expire,
	)

	return nil
}
