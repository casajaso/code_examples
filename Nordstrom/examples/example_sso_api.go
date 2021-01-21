/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
golang example using custom credential_process to retrieve credentials
Requires or aws-sdk v1.16.0 above to support custom credential_process
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/mitchellh/go-homedir"
)

type ssoCache struct {
	StartUrl    string `json: "startUrl"`
	Region      string `json: "region"`
	AccessToken string `json: "accessToken"`
	ExpiresAt   string `json: "expiresAt"`
}

var cache ssoCache
var creds *sso.RoleCredentials
var awsSession *session.Session

func getSSOCache() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	file := filepath.Join(home, "/.aws/sso/cache/0c2eea704ee08603d192a6e22220cb618db79d17.json")
	rawdata, err := os.Open(file)
	if err != nil {
		return "", err
	}
	jsonParser := json.NewDecoder(rawdata)
	if err = jsonParser.Decode(&cache); err != nil {
		fmt.Println(fmt.Errorf("parsing SSO cache: %s", err.Error()))
	}
	accessToken := cache.AccessToken
	return accessToken, nil
}

func getCredentials() error {
	accessToken, err := getSSOCache()
	if err != nil {
		return err
	}
	accountId := "116019673048"
	roleName := "AdministratorAccess"
	s, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:                        aws.String("us-west-2"),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	})
	if err != nil {
		return err
	}
	svc := sso.New(s)
	// svc := sso.New(awsSession, aws.NewConfig().WithRegion("us-west-2"))
	providorInput := &sso.GetRoleCredentialsInput{
		AccessToken: &accessToken,
		AccountId:   &accountId,
		RoleName:    &roleName,
	}
	c, err := svc.GetRoleCredentials(providorInput)
	if err != nil {
		return err
	}
	creds = c.RoleCredentials
	// jsonParser := json.NewDecoder(c)
	// if err = jsonParser.Decode(&creds); err != nil {
	// 	fmt.Println(fmt.Errorf("parsing SSO cache: %s", err.Error()))
	// }
	return nil
}

func main() {
	err := getCredentials()
	if err != nil {
		fmt.Println(fmt.Errorf("Error: (%s)", err))
	}
	awsSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-west-2"),
			Credentials: credentials.NewStaticCredentials(
				*creds.AccessKeyId,
				*creds.SecretAccessKey,
				*creds.SessionToken,
			),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	})
	svc := sts.New(awsSession)
	input := &sts.GetCallerIdentityInput{}
	result, err := svc.GetCallerIdentity(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println(result)

}
