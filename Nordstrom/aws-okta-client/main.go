/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Authored by Cloud Engineering <cloudengineering@nordstrom.com>

Copyright 2018 @ Nordstrom, Inc. All rights reserved.
*/

package main

import (
	"gitlab.nordstrom.com/public-cloud/aws-okta/cmd"
)

var (
	// VERSION is set during build
	Version = "0.0"
)

func main() {
	cmd.Execute(Version)
}
