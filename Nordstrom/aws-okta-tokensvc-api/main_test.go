/*
AWS Okta Token Service -- https://gitlab.nordstrom.com/public-cloud/aws-okta-token-service
Authored by Cloud Engineering <cloudengineering@nordstrom.com>

Copyright 2018 @ Nordstrom, Inc. All rights reserved.
*/

package main

import (
	"testing"

	"gitlab.nordstrom.com/public-cloud/aws-okta-token-service/lib"
	//"gitlab.nordstrom.com/public-cloud/aws-okta-token-service/lib"
)

func TestContains(t *testing.T) {
	groups := []string{"AWS#NORD-SANDBOXTEAM01-DevUsers-Team#468543742856", "AWS#NORD-SANDBOXTEAM02-DevUsers-Team#468543742856"}

	containTrue := lib.Contains(groups, "AWS#NORD-SANDBOXTEAM02-DevUsers-Team#468543742856")
	if containTrue == false {
		t.Errorf("contains was incorrect, got: %T , want: %T", containTrue, false)
	}

	containFalse := lib.Contains(groups, "AWS#NORD-ROLE-NOT-IN-LIST#468543742856")
	if containFalse == true {
		t.Errorf("contains was incorrect, got: %T , want: %T", containFalse, true)
	}
}

func TestCreateRoleArnsFromGroups(t *testing.T) {
	arn := "arn:aws:iam::468543742856:role/NORD-SANDBOXTEAM01-DevUsers-Team"
	groups := []string{"AWS#NORD-SANDBOXTEAM01-DevUsers-Team#468543742856", "AWS#NORD-SANDBOXTEAM02-DevUsers-Team#468543742856"}

	roleGroups := lib.GroupsToARNs(groups)
	if !(roleGroups[0] == arn) {
		t.Errorf("createRoleArnsFromGroups was incorrect, got: %s , want: %s", roleGroups[0], arn)
	}
}

func TestRoleArnRegex(t *testing.T) {
	validArn := "arn:aws:iam::468543742856:role/NORD-SANDBOXTEAM01-DevUsers-Team"

	if !(lib.ValidRole.MatchString(validArn)) {
		t.Errorf("validRole was incorrect, got: %T , want: %T", true, false)
	}

	inValidArn := "NORD-SANDBOXTEAM01-DevUsers-Team"

	if lib.ValidRole.MatchString(inValidArn) {
		t.Errorf("validRole was incorrect, got: %T , want: %T", true, false)
	}
}

// func TestCreateUniqueRoleSessionName(t *testing.T) {
// 	sessionName := "okta.session"
// 	sub := "00umqxus8TROl4PPS2p6"
// 	expectedResult := "00umqxus8TROl4PPS2p6-okta.session"
//
// 	uniqueName := lib.UniqueSessionName(sub, sessionName)
//
// 	if !(uniqueName == expectedResult) {
// 		t.Errorf("createUniqueRoleSessionName was incorrect, got: %s , want: %s", uniqueName, expectedResult)
// 	}
//
// 	// sessionName passed is 69 characters
// 	sessionName = "okta.session-really-really-really-long-length-more-than-65-characters"
// 	sub = "00umqxus8TROl4PPS2p6"
// 	// expected result is truncated 65 character string
// 	expectedResult = "00umqxus8TROl4PPS2p6-okta.session-really-really-really-long-lengt"
//
// 	uniqueName = lib.UniqueSessionName(sub, sessionName)
//
// 	if !(len(uniqueName) == 65) {
// 		t.Errorf("createUniqueRoleSessionName length was incorrect, got: %d , want: %d", len(uniqueName), 65)
// 	}
//
// 	if !(uniqueName == expectedResult) {
// 		t.Errorf("createUniqueRoleSessionName was incorrect, got: %s , want: %s", uniqueName, expectedResult)
// 	}
//
// }
