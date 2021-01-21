/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"time"

	"github.com/99designs/keyring"
	ini "gopkg.in/ini.v1"
)

// [CUSTOMIZABLE SITE-SPECIFIC VARIABLES] - START
//IDS (pci, nonpci) oidc clientids pool
const (
	Organization       = "nordstrom"
	ProdPersistant     = false // set true to lock to prod endpoints
	CredProcessVersion = 1
	AWSSessionName     = "AWSSession"
	OktaSessionName    = "OktaSession"
	AppName            = "aws-okta"
	logformat          = "prefixed"
)

// os/user env vars to unset SEE: unset in execute providor
var ClearVars = []string{
	"AWS_CONFIG_FILE",
	"AWS_SHARED_CREDENTIALS_FILE",
	"AWS_DEFAULT_PROFILE",
	"AWS_PROFILE",
	"AWS_OKTA_PROFILE",
	"AWS_ROLE_SESSION_NAME",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_SECURITY_TOKEN",
	"AWS_SDK_LOAD_CONFIG",
}

// config/credentials perams to delete SEE: parse in profile providor
var RemoveKeys = []string{
	"source_profile",
	"aws_access_key_id",
	"aws_secret_access_key",
	"aws_session_token",
}

var Platform = runtime.GOOS

// Okta Session keyring item-name
var (
	log   *Logger
	Stage string
)

var CID = map[string]map[string]map[string]string{
	"prod": {
		"pci": {
			"clientid": "0oa53ji9f4L4ErE5O2p7",
		},
		"nonpci": {
			"clientid": "0oa28xw7eaSoJF6NK2p7",
		},
	},
	"dev": {
		"pci": {
			"clientid": "0oanju1rkiIAIAoPg0h7",
		},
		"nonpci": {
			"clientid": "0oagdckynkMFS57ZY0h7",
		},
	},
}

var DUR = map[string]map[string]map[string]time.Duration{
	"pci": {
		"oidc": {
			"laps": (15 * time.Minute),
		},
		"aws": {
			"min":     (15 * time.Minute),
			"max":     (15 * time.Minute),
			"default": (15 * time.Minute),
		},
	},
	"nonpci": {
		"oidc": {
			"laps": (4 * time.Hour),
		},
		"aws": {
			"min":     (15 * time.Minute),
			"max":     (12 * time.Hour),
			"default": (1 * time.Hour),
		},
	},
}

var EPS = map[string]map[string]*url.URL{
	"prod": {
		"oktaserverus": &url.URL{
			Scheme: "https",
			Host:   Organization + ".okta.com",
		},
		"token": &url.URL{
			Scheme: "https",
			Host:   "tokensvc.prod.cloud.vip.nordstrom.com",
			Path:   "v2",
		},
	},
	"dev": {
		"oktaserverus": &url.URL{
			Scheme: "https",
			Host:   Organization + ".oktapreview.com",
		},
		"token": &url.URL{
			Scheme: "https",
			Host:   "tokensvc.dev.cloud.vip.nordstrom.com",
			Path:   "v2",
		},
	},
	"shared": {
		"redirect": &url.URL{
			Scheme: "http",
			Host:   "localhost:5556",
			Path:   "/auth/okta/callback",
		},
		"success": &url.URL{
			Scheme: "https",
			Host:   "s3-us-west-2.amazonaws.com",
			Path:   "/aws-auth/success/index.html",
		},
		"failure": &url.URL{
			Scheme: "https",
			Host:   "s3-us-west-2.amazonaws.com",
			Path:   "/aws-auth/success/index.html",
		},
		"latest": &url.URL{
			Scheme: "https",
			Host:   "storage.googleapis.com",
			Path:   "/aws-okta/release/stable.json",
		},
		"faq": &url.URL{
			Scheme: "https",
			Host:   "confluence.nordstrom.net",
			Path:   "/display/PubCloud/aws-okta",
		},
		"repo": &url.URL{
			Scheme: "https",
			Host:   "gitlab.nordstrom.com",
			Path:   "public-cloud/aws-okta",
		},
	},
	"darwin": {
		"download": &url.URL{
			Scheme: "https",
			Host:   "storage.googleapis.com",
			Path:   "/aws-okta/release/LATEST/bin/darwin/x64/aws-okta",
		},
	},
	"linux": {
		"download": &url.URL{
			Scheme: "https",
			Host:   "storage.googleapis.com",
			Path:   "/aws-okta/release/LATEST/bin/linux/x64/aws-okta",
		},
	},
	"windows": {
		"download": &url.URL{
			Scheme: "https",
			Host:   "storage.googleapis.com",
			Path:   "/aws-okta/release/LATEST/bin/win/x64/aws-okta.exe",
		},
	},
}

// [CUSTOMIZABLE SITE-SPECIFIC VARIABLES] - END

// Options data
type ProvidorOptions struct {
	SessionDuration    time.Duration
	AssumeRoleDuration time.Duration
	ExpiryWindow       time.Duration
	Profiles           map[string]map[string]string
	AssumeRoleArn      string
	ProfileName        string
	ProfileSection     *ini.Section
	Backends           []keyring.BackendType
	Keyring            keyring.Keyring
	ClientID           string
	Stage              string
	Environment        string
	LogLevel           string
	Config             *Config
	OIDC               *OIDC
}

type OIDC struct {
	Organization string
	ClientID     string
	OktaServerUS *url.URL
	TokenSVC     *url.URL
	SessionLaps  time.Duration
	Redirect     *url.URL
	Success      *url.URL
	Failure      *url.URL
	Token        string
}

type AWS struct {
	SessionMin     time.Duration
	SessionMax     time.Duration
	SessionDefault time.Duration
}

//BaseConfig stores os/env/stage specific endpoint and session duration values
type Config struct {
	LogConfig   *LogConfig
	Stage       string
	Environment string
	Profile     string
	OIDC        *OIDC
	AWS         *AWS
}

func NewBaseConfig(o ProvidorOptions) (*Config, error) {

	if ProdPersistant { //if true lock to prod enpoints
		Stage = "prod"
	}

	logConfig := &LogConfig{
		Format: logformat,
		Level:  o.LogLevel,
	}

	logger, err := NewLogProvidor(logConfig)
	if err != nil {
		fmt.Sprint(err)
	}

	log = logger

	c := make(map[string]interface{})
	cfg := &Config{}

	c["LogConfig"] = logConfig
	c["Stage"] = o.Stage
	c["Environment"] = o.Environment
	c["Backend"] = keyring.AvailableBackends()
	c["OIDC"] = map[string]interface{}{
		"Organization": Organization,
		"ClientID":     CID[o.Stage][o.Environment]["clientid"],
		"OktaServerUS": EPS[o.Stage]["oktaserverus"],
		"TokenSVC":     EPS[o.Stage]["token"],
		"Redirect":     EPS["shared"]["redirect"],
		"Success":      EPS["shared"]["success"],
		"Failure":      EPS["shared"]["failure"],
		"SessionLaps":  DUR[o.Environment]["oidc"]["laps"],
	}
	c["AWS"] = map[string]interface{}{
		"SessionMin":     DUR[o.Environment]["aws"]["min"],
		"SessionMax":     DUR[o.Environment]["aws"]["max"],
		"SessionDefault": DUR[o.Environment]["aws"]["default"],
	}

	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
