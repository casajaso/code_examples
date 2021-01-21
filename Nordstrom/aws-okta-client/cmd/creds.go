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
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
)

// credsCmd represents the creds command
var credsCmd = &cobra.Command{
	Use:     "creds -p <profile-name>",
	Aliases: []string{"list"},
	Short:   "Retrieve authorized AWS roles and retrieve credentials for selected role",
	PreRunE: credsPre,
	RunE:    credsRun,
}

var (
	switchRole bool
	execName   string
)

func init() {
	RootCmd.AddCommand(credsCmd)
	credsCmd.Flags().StringVarP(&role, "role-arn", "r", "", "Set role_arn for target AWS federated role - bypasses role selection prompt")
	credsCmd.Flags().StringVarP(&ProfileName, "profile-name", "p", "nordstrom-federated", "Set profile name")
	credsCmd.Flags().DurationVarP(&assumeRoleTTL, "assume-role-ttl", "t", 0, "Set AWS Session duration. Options [NonPCI: 0h15m0s - 1h0m0s, PCI: 0h15m0s]")
}

func credsPre(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("role-arn") {
		roleFlagSet = true
	}
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

func credsRun(cmd *cobra.Command, args []string) error {

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

	if len(OktaSessionKey) <= len(lib.OktaSessionName) {
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

	if !token.SessionToken.Valid() {
		log.Infof("Refreshing Okta session token - expired: %s", time.Until(token.SessionToken.Expiry))

		err := lib.RefreshOktaSessionToken(Keyring, item, cfg)
		if err != nil {
			log.Errorf("Unable to refresh Okta Session Credentials: (%s)", err)
		}

		item, err := Keyring.Get(OktaSessionKey)
		if err != nil {
			log.Error(err)
			return err
		}

		if err = json.Unmarshal(item.Data, token); err != nil {
			log.Error(err)
			return err
		}
	}

	cfg.OIDC.Token = token.IDToken

	cfgFile, err := lib.NewProfileProvider("config")
	if err != nil {
		log.Error(err)
		return err
	}

	credFile, err := lib.NewProfileProvider("credentials")
	if err != nil {
		log.Error(err)
		return err
	}

	prf, err := cfgFile.Parse(ProfileName, false)
	if err != nil {
		log.Error(err)
		return err
	}

	_, err = credFile.Parse(ProfileName, false)
	if err != nil {
		log.Error(err)
		return err
	}

	err = credFile.Clear()
	if err != nil {
		log.Error(err)
		return err
	}

	g, err := lib.NewGroupsProvider(cfg)
	if err != nil {
		log.Error(err)
		return err
	}

	roleGroups, err := g.RetrieveGroups()
	if err != nil {
		log.Error(err)
		return err
	}

	if len(roleGroups) == 0 {
		log.Errorf("Unable to retrieve role(s); AD group membership must include access to at least one (%s) federated role",
			cfg.Environment,
		)
		return err
	}

	ra, _ := prf.GetKey("_role_arn")

	r, err := lib.SelectRole(role, roleGroups)
	if err != nil {
		log.Errorf("Unable to set role selection: (%s)", err)
		return err
	}

	if ra != nil {
		if ra.String() != r {
			switchRole = true
		}
	}

	var assumeRoleDuration time.Duration

	if assumeRoleTTL > 0 {
		assumeRoleDuration = assumeRoleTTL
	} else {
		assumeRoleDuration = cfg.AWS.SessionDefault
	}

	if runtime.GOOS == "windows" {
		execName = fmt.Sprintf("%s.exe", os.Args[0])
	} else {
		execName = os.Args[0]
	}

	credProc := fmt.Sprintf("%s cproc --profile-name %s --assume-role-ttl %s", execName, ProfileName, assumeRoleDuration)

	cfgKeys := map[string]string{
		"_role_arn":          r,
		"credential_process": credProc,
	}

	credKeys := map[string]string{
		"_provider":          os.Args[0],
		"credential_process": credProc,
	}

	log.Infof("Updating: [profile %s]",
		ProfileName,
	)

	if err := cfgFile.Update(cfgKeys); err != nil {
		log.Error(err)
		return err
	}

	if err := credFile.Update(credKeys); err != nil {
		log.Error(err)
		return err
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

	if switchRole {
		p.DeleteAWSSession()
	}

	_, _, err = p.Retrieve(prf)
	if err != nil {
		log.Error(err)
		return err
	}

	fmt.Fprintf(os.Stdout, "\nYou can now run AWS CLI using (%s) credentials when specifying option: `--profile %s`\n\t\tExample: `aws sts get-caller-identity --profile %s`",
		strings.Split(r, ":")[5][5:],
		ProfileName,
		ProfileName,
	)

	return nil
}
