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
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	lambdaSVC "github.com/aws/aws-sdk-go/service/lambda"
)

var (
	ValidRole = regexp.MustCompile("arn:aws:iam::\\d{12}:role/?[a-zA-Z_0-9+=,.@\\-_/]+")
)

func GetADGroups(callerEmail string, AskLDAP string) ([]string, error) {

	type Input struct {
		Query string `json:"query"`
		User  string `json:"user"`
	}

	type Body struct {
		Groups []string `json:"groups"`
	}

	type Response struct {
		Payload Body `json:"result"`
	}

	var reduce []string

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambdaSVC.New(sess, &aws.Config{Region: aws.String(ldapRegion)})

	input := Input{"list_user_access", callerEmail}

	b, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("Unable to format LDAP query: (%s)", err)
	}

	result, err := client.Invoke(&lambdaSVC.InvokeInput{FunctionName: aws.String(AskLDAP), Payload: b})
	if err != nil {
		return nil, fmt.Errorf("Unable to reach LDAP_Lambda: (%s)", err)
	}

	if *result.StatusCode != 200 {
		err := fmt.Errorf("Unable to request AD Security-groups: (%s)", *result.StatusCode)
		return nil, err
	}

	if len(result.Payload) == 0 {
		err := fmt.Errorf("Unable find AD Security groups-for: (%s)", callerEmail)
		return nil, err
	}

	var resp Response

	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		return nil, err
	}

	reduce = Filter(resp.Payload.Groups, func(group string) bool {
		return strings.Contains(group, "AWS#")
	})

	return reduce, nil
}

// GroupsToARNs constructs iam role arns from ad groups
func GroupsToARNs(groups []string) []string {

	var roleGroups []string

	for _, group := range groups {
		s := strings.Split(group, "#")
		_, roleName, accountNumber := s[0], s[1], s[2]
		roleGroups = append(roleGroups, fmt.Sprintf("arn:aws:iam::%s:role/%s", accountNumber, roleName))
	}

	return roleGroups
}

// GetEnvGroups filters role arns by target environment
func GetEnvGroups(env string, groups []string) ([]string, error) {

	var envGroups []string

	for _, group := range groups {
		s := strings.Split(group, "#")
		roleName := s[1]

		if env == "pci" {
			if strings.Contains(roleName, pciFilter) {
				envGroups = append(envGroups, group)
			}
		}

		if env == "nonpci" {
			if !strings.Contains(roleName, pciFilter) {
				envGroups = append(envGroups, group)
			}
		}
	}
	return envGroups, nil
}

//filter function
func Filter(groups []string, check func(string) bool) []string {

	awsGroups := make([]string, 0)

	for _, group := range groups {
		if check(group) {
			awsGroups = append(awsGroups, group)
		}
	}

	return awsGroups
}
