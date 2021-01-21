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
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
)

// consoleCmd represents the console command
var consoleCmd = &cobra.Command{
	Use:    "console -p <profile-name>",
	Short:  "Open the AWS console in a browser",
	PreRun: consolePre,
	RunE:   consoleRun,
}

func init() {
	RootCmd.AddCommand(consoleCmd)
	consoleCmd.Flags().BoolVarP(&printUrl, "stdout", "s", false, "Display AWS console sign-on token")
	consoleCmd.Flags().StringVarP(&ProfileName, "profile-name", "p", "nordstrom-federated", "Set profile name")
	consoleCmd.Flags().DurationVarP(&assumeRoleTTL, "assume-role-ttl", "t", 0, "Set AWS Session duration. Options [NonPCI: 0h15m0s - 1h0m0s, PCI: 0h15m0s]")
}

func consolePre(cmd *cobra.Command, args []string) {

	if cmd.Flags().Changed("stdout") {
		printnUrlFlagSet = true
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

	if !profileFlagSet {
		if len(args) > 0 {
			ProfileName = args[0]
		}
	}
}

func consoleRun(cmd *cobra.Command, args []string) error {

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

	creds, _, err := p.Retrieve(prf)
	if err != nil {
		log.Error(err)
		return err
	}

	jsonBytes, err := json.Marshal(map[string]string{
		"sessionId":    creds.AccessKeyID,
		"sessionKey":   creds.SecretAccessKey,
		"sessionToken": creds.SessionToken,
	})
	if err != nil {
		log.Error(err)
		return err
	}

	req, err := http.NewRequest("GET", "https://signin.aws.amazon.com/federation", nil)
	if err != nil {
		log.Error(err)
		return err
	}

	q := req.URL.Query()

	q.Add("Action", "getSigninToken")

	q.Add("Session", string(jsonBytes))

	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Call to getSigninToken failed with %v", resp.Status)
		log.Error(err)
		return err
	}

	var respParsed map[string]string
	if err = json.Unmarshal([]byte(body), &respParsed); err != nil {
		log.Error(err)
		return err
	}

	signinToken, ok := respParsed["SigninToken"]
	if !ok {
		log.Error(err)
		return err
	}

	region, _ := prf.GetKey("region")

	destination := "https://console.aws.amazon.com/"

	if region != nil {
		destination = fmt.Sprintf(
			"https://%s.console.aws.amazon.com/console/home?region=%s",
			region.String(), region.String(),
		)
	}

	loginURL := fmt.Sprintf(
		"https://signin.aws.amazon.com/federation?Action=login&Issuer=aws-okta&Destination=%s&SigninToken=%s",
		url.QueryEscape(destination),
		url.QueryEscape(signinToken),
	)

	if printnUrlFlagSet {
		fmt.Printf("AWS sign-on token for: %s\n", strings.Split(prf.Key("_role_arn").String(), ":")[5][5:])
		fmt.Println(loginURL)
	} else {
		fmt.Printf("Opening AWS Console for: %s", strings.Split(prf.Key("_role_arn").String(), ":")[5][5:])
		if err = browser.OpenURL(loginURL); err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}
