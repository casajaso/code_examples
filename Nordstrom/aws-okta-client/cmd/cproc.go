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

type credProcess struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
}

// cprocCmd represents the cproc command (hidden)
var cprocCmd = &cobra.Command{
	Use:     "cproc =p <profile-name>",
	Short:   "Formats and presents session credentials to AWS API calls using process-credentials or AWS credential-process",
	Hidden:  true,
	PreRunE: cprocPre,
	RunE:    cprocRun,
}

func init() {
	RootCmd.AddCommand(cprocCmd)
	cprocCmd.Flags().StringVarP(&ProfileName, "profile-name", "p", "nordstrom-federated", "Set profile name")
	cprocCmd.Flags().DurationVarP(&assumeRoleTTL, "assume-role-ttl", "t", 0, "Set AWS Session duration. Options [NonPCI: 0h15m0s - 1h0m0s, PCI: 0h15m0s]")
}

func cprocPre(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("role-arn") {
		roleFlagSet = true
	}

	if cmd.Flags().Changed("profile-name") {
		profileFlagSet = true
	}

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

func cprocRun(cmd *cobra.Command, args []string) error {

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

	creds, exp, err := p.Retrieve(prf)
	if err != nil {
		log.Error(err)
		return err
	}

	cp := credProcess{
		Version:         lib.CredProcessVersion,
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Expiration:      (exp).Format(time.RFC3339),
	}

	var output []byte

	output, err = json.MarshalIndent(cp, "", "    ")
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, string(output))

	return nil
}
