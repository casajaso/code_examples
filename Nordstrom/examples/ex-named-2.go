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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var awsSession *session.Session

// func init() {
// 	s, err := session.NewSession(&aws.Config{})
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	awsSession = s
// }

func sessionFromProfileDefault(region string, verbose bool) error {
	s, err := session.NewSession(
		&aws.Config{
			Region:                        aws.String(region),
			CredentialsChainVerboseErrors: aws.Bool(verbose),
		},
	)
	if err != nil {
		return err
	}
	awsSession = s
	return nil
}

func sessionFromProfileName(profile string, region string, verbose bool) error {
	s, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Profile:           profile,
		Config: aws.Config{
			Region:                        aws.String(region),
			CredentialsChainVerboseErrors: aws.Bool(verbose),
		},
	})
	if err != nil {
		return err
	}
	awsSession = s
	return nil
}

func main() {
	// err := sessionFromProfileDefault("us-west-2", false) //for default profile
	err := sessionFromProfileName("nordstrom-federated", "us-west-2", true) //for named profiles
	if err != nil {
		fmt.Println(err.Error())
	}
	svc := s3.New(awsSession)
	input := &s3.ListBucketsInput{}

	result, err := svc.ListBuckets(input)
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

// sess, err := session.NewSession(&aws.Config{
//     Region:      aws.String("us-west-2"),
//     Credentials: credentials.NewSharedCredentials("", "test-account"),
// })

// _, err := sess.Config.Credentials.Get()
