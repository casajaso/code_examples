package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	s, err := session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Profile:           "nordstrom-federated", //omit or "default" if using default profile
			Config: aws.Config{
				Region:                        aws.String("us-west-2"),
				CredentialsChainVerboseErrors: aws.Bool(true),
			},
		},
	)
	if err != nil {
		fmt.Println(err.Error())
	}

	svc := s3.New(s)

	resp, err := svc.ListBuckets(
		&s3.ListBucketsInput{},
	)
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
	fmt.Println(resp)
}
