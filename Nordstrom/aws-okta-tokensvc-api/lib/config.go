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
	"time"
)

//IDS (pci, nonpci) oidc clientids pool
const (
	Organization = "nordstrom"
	pciFilter    = "PCI"
	ldapRegion   = "us-west-2"
	LogFormat    = "json"
	LogLevel     = "debug"
)

var log *Logger

var CID = map[string]map[string]map[string]string{
	"prod": {
		"pci": {
			"clientid": "0oa53ji9f4L4ErE5O2p7",
		},
		"nonpci": {
			"clientid": "0oa28xw7eaSoJF6NK2p7",
		},
		"groups": {
			"AskLDAP": "cloud-api-ldap-prod-listCloudGroups",
		},
	},
	"dev": {
		"pci": {
			"clientid": "0oanju1rkiIAIAoPg0h7",
		},
		"nonpci": {
			"clientid": "0oagdckynkMFS57ZY0h7",
		},
		"groups": {
			"AskLDAP": "cloud-api-ldap-dev-listCloudGroups",
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
	},
	"dev": {
		"oktaserverus": &url.URL{
			Scheme: "https",
			Host:   Organization + ".oktapreview.com",
		},
	},
}

// ProviderOptions data
type ProvidorOptions struct {
	Stage       string
	Environment string
	LogLevel    string
}

type OIDC struct {
	Organization string
	ClientID     string
	OktaServerUS *url.URL
}

//Limits stores session duration limits
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
	AskLDAP     string
	OIDC        *OIDC
	AWS         *AWS
}

func NewBaseConfig(o ProvidorOptions) (*Config, error) {

	logConfig := &LogConfig{
		Format: LogFormat,
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
	c["AskLDAP"] = CID[o.Stage]["groups"]["AskLDAP"]
	c["OIDC"] = map[string]interface{}{
		"Organization": Organization,
		"ClientID":     CID[o.Stage][o.Environment]["clientid"],
		"OktaServerUS": EPS[o.Stage]["oktaserverus"],
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
