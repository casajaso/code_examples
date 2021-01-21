/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"gitlab.nordstrom.com/public-cloud/aws-okta-token-service/handler"
)

func main() {

	fmt.Println("Starting Handler")

	lambda.Start(handler.Handler)
}
