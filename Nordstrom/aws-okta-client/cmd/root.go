/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/spf13/cobra"
	"gitlab.nordstrom.com/public-cloud/aws-okta/lib"
	"gopkg.in/ini.v1"
)

const ()

var (
	ErrCommandMissing              = errors.New("must specify command to run")
	ErrTooManyArguments            = errors.New("too many arguments")
	ErrTooFewArguments             = errors.New("too few arguments")
	ErrFailedToSetCredentials      = errors.New("Failed to store credentials in keyring")
	ErrFailedToValidateCredentials = errors.New("Failed to validate credentials")
)

// argument flags and variables
var (
	Version              string
	backend              string
	Backends             []keyring.BackendType
	osBackends           []string
	logLevel             string
	log                  *lib.Logger
	dev                  bool
	pci                  bool
	stage                string
	environment          string
	Config               *lib.Config
	assumeRoleTTL        time.Duration
	assumeRoleTTLFlagSet bool
	AssumeRoleDuration   time.Duration
	role                 string
	roleFlagSet          bool
	ProfileName          string
	ProfileSection       *ini.Section
	profileFlagSet       bool
	printUrl             bool
	printnUrlFlagSet     bool
	install              bool
	installFlagSet       bool
	Credentials          *credentials.Value
	download             bool
	downloadFlagSet      bool
	credsForPS           bool
	credsForPSFlagSet    bool
	Keyring              keyring.Keyring
	showExpiration       bool
)

// Execute adds all child commands to the root command sets top-level variables and flags.
// This is called by main.main().
func Execute(version string) {
	fmt.Fprintf(os.Stdout, "\n")
	Version = version
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		switch err {
		case ErrTooFewArguments, ErrTooManyArguments:
			RootCmd.Usage()
		}
		os.Exit(1)
	}
}

var CMDName = os.Args[0]

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:               CMDName,
	Short:             fmt.Sprintf("\n%s Enables AWS authentication with Okta SSO/MFA", CMDName),
	SilenceUsage:      true,
	SilenceErrors:     true,
	PersistentPreRunE: prerunE,
	PersistentPostRun: postrun,
}

func init() {

	for _, backendType := range keyring.AvailableBackends() {
		osBackends = append(osBackends, string(backendType))
	}

	RootCmd.PersistentFlags().StringVarP(&backend, "backend", "b", osBackends[0], fmt.Sprintf("Set backend/credential-store. Options: %s", osBackends))
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", "warn", "Set verbocity/log-level. Options: [warn, info, debug]")
}

func prerunE(cmd *cobra.Command, args []string) error {

	if !cmd.Flags().Lookup("backend").Changed {
		backendFromEnv, ok := os.LookupEnv("AWS_OKTA_BACKEND")
		if ok {
			backend = backendFromEnv
		}
	}

	switch strings.ToLower(logLevel) {
	case "":
		logLevel = "warn"
	case "warn":
		logLevel = "warn"
	case "info":
		logLevel = "info"
	case "debug":
		logLevel = "debug"
	default:
		fmt.Errorf("%s: (-v/verbosity must be one of: warn, info, debug)", ErrTooFewArguments)
		os.Exit(1)
	}

	if dev {
		stage = "dev"
	} else {
		stage = "prod"
	}

	if pci {
		environment = "pci"
	} else {
		environment = "nonpci"
	}

	if backend != "" {
		Backends = append(Backends, keyring.BackendType(backend))
	}

	kr, err := lib.NewKeyringSession(Backends)
	if err != nil {
		log.Error(err)
		return err
	}

	Keyring = kr

	cfgOps := &lib.ProvidorOptions{
		Stage:       stage,
		Environment: environment,
		LogLevel:    logLevel,
	}

	cfg, err := lib.NewBaseConfig(*cfgOps)
	if err != nil {
		log.Errorf("Unable to set BaseConfig: (%s)", err)
	}

	logger, err := lib.NewLogProvidor(cfg.LogConfig)
	if err != nil {
		log.Error(err)
	}

	log = logger

	// u, err := lib.UpdateHandler(Version)
	// if err != nil {
	// 	log.Warn(err)
	// }

	// resp, od, err := u.Check()
	// if err != nil {
	// 	log.Warn(err)
	// }

	// if od {
	// 	fmt.Fprint(os.Stdout, "\n")
	// 	if strings.Contains(resp, "REQUIRED") { //forces exit if out of date and update dialog includes string `REQUIRED`
	// 		log.Errorf("*Required* %s", resp)
	// 	} else {
	// 		log.Warnf("%s", resp)
	// 	}
	// }

	return nil

}

func postrun(cmd *cobra.Command, args []string) {
	fmt.Fprintf(os.Stdout, "\n")
	os.Exit(0)

}

func loadDurationFlagFromEnv(cmd *cobra.Command, flagName string, envVar string, val *time.Duration, flagSet *bool) error {

	if cmd.Flags().Lookup(flagName).Changed {
		return nil
	}

	fromEnv, ok := os.LookupEnv(envVar)
	if !ok {
		return nil
	}

	dur, err := time.ParseDuration(fromEnv)
	if err != nil {
		log.Debugf(fmt.Sprint(err))
		return nil
	}

	*flagSet = true
	*val = dur
	return nil
}

func loadStringFlagFromEnv(cmd *cobra.Command, flagName string, envVar string, val *string, flagSet *bool) error {

	if cmd.Flags().Lookup(flagName).Changed {
		return nil
	}

	fromEnv, ok := os.LookupEnv(envVar)
	if !ok {
		return nil
	}

	*flagSet = true

	*val = fromEnv

	return nil
}
