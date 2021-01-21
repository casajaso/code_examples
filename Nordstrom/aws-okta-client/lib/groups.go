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
	"net/http"
	"strings"
)

// GroupsProvider data
type GroupsProvider struct {
	Config *Config
}

type config *Config

var groups []string

//NewGroupsProvider inits new groups config
func NewGroupsProvider(config *Config) (*GroupsProvider, error) {
	return &GroupsProvider{
		Config: config,
	}, nil
}

//RetrieveGroups hits api to grab groups
func (p *GroupsProvider) RetrieveGroups() ([]string, error) {

	authAuthenticatorURL := p.Config.OIDC.TokenSVC.String()

	client := &http.Client{}

	req, err := http.NewRequest("POST", authAuthenticatorURL, nil)
	req.Header.Add("Authorization", p.Config.OIDC.Token)
	req.Header.Add("Stage", p.Config.Stage)
	req.Header.Add("Environment", p.Config.Environment)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Caller", "groups")
	if err != nil {
		return nil, fmt.Errorf("Failed to build post request: (%s)", err)
	}

	var postErr error

	resp, err := client.Do(req)
	if err != nil {
		postErr = err
	}

	if resp != nil {

		defer resp.Body.Close()

		var result map[string]interface{}

		json.NewDecoder(resp.Body).Decode(&result)

		log.Infof("Requesting roles - Client-RequestId: [%s]", resp.Header.Get("x-amzn-RequestId"))

		if resp.StatusCode >= 400 || postErr != nil {
			return nil, fmt.Errorf(result["message"].(string))
		}

		b, err := json.Marshal(result["Groups"])
		if err != nil {
			return nil, err
		}

		jsonErr := json.Unmarshal(b, &groups)
		if jsonErr != nil {
			return nil, fmt.Errorf("Failed to marshal/unmarshal data: (%s)", jsonErr)
		}
		return groups, nil
	}
	return nil, postErr
}

// ConstructRoleArns returns AWS IAM Role ARNs from AD Security Groups
func ConstructRoleArns(groups []string) []string {
	var roleGroups []string
	for _, group := range groups {
		s := strings.Split(group, "#")
		_, roleName, accountNumber := s[0], s[1], s[2]
		roleGroups = append(roleGroups, fmt.Sprintf("arn:aws:iam::%s:role/%s", accountNumber, roleName))
	}
	return roleGroups
}
