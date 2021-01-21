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
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:                   "exec -p <profile-name> [flags] -- <command syntax>",
	Short:                 "Run a command in an AWS authenticated sub-shell",
	PreRunE:               execPre,
	RunE:                  execRun,
	DisableFlagsInUseLine: true,
}

var (
	CommandSegment []string
	commandArgs    []string
	command        string
	dashIx         int
)

func init() {
	RootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&ProfileName, "profile-name", "p", "nordstrom-federated", "Set profile name")
	execCmd.Flags().DurationVarP(&assumeRoleTTL, "assume-role-ttl", "t", 0, "Set AWS Session duration. Options [NonPCI: 0h15m0s - 1h0m0s, PCI: 0h15m0s]")
	// execCmd.MarkFlagRequired(ProfileName)
}

func execPre(cmd *cobra.Command, args []string) error {

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

	dashIx = cmd.ArgsLenAtDash()
	CommandSegment = args[dashIx:]

	if profileFlagSet {
		ProfileName = ProfileName
	} else if len(args[:dashIx]) != 0 {
		ProfileName = args[0]
	} else {
		ProfileName = ProfileName
	}

	return nil
}

func execRun(cmd *cobra.Command, args []string) error {

	if len(CommandSegment) < 1 {
		log.Error(ErrCommandMissing)
		return ErrCommandMissing
	}

	if lib.Platform == "windows" {
		command = "cmd"
		if len(CommandSegment) > 1 {
			commandArgs = append([]string{"/C"}, CommandSegment[0:]...)
		} else {
			commandArgs = append([]string{"/C"}, CommandSegment[0])
		}
	} else {
		command = CommandSegment[0]
		if len(CommandSegment) > 1 {
			commandArgs = CommandSegment[1:]
		}
	}

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

	var rgn string
	region, _ := prf.GetKey("region")
	if region != nil {
		rgn = region.String()
	}

	ep := lib.NewExecProvider(prf.Name(), creds, rgn)

	ecmd := exec.Command(command, commandArgs...)

	ecmd.Stdin = os.Stdin

	ecmd.Stdout = os.Stdout

	ecmd.Stderr = os.Stderr

	ecmd.Env = ep.Env

	// Forward SIGINT, SIGTERM, SIGKILL to the child command
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt, os.Kill)

	go func() {
		sig := <-sigChan
		if ecmd.Process != nil {
			ecmd.Process.Signal(sig)
		}
	}()

	var waitStatus syscall.WaitStatus

	if err := ecmd.Run(); err != nil {
		if err != nil {
			log.Error(err)
			return err
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			os.Exit(waitStatus.ExitStatus())
		}
	}

	return nil
}
