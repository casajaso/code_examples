/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"fmt"
	"time"
)

// createUniqueRoleSessionName combines Okta's user identifier (sub claim) and Role Session Name
func UniqueSessionName(sub string, email string) string {
	var safeSubstring string

	// to make RoleSessionName unique we prefix the Sub unique identitifer
	// e.g., "<username>@nordstrom.com-okta.session"
	// value := fmt.Sprintf("%s-%s", sub, email)
	value := fmt.Sprintf("%s", email)

	// Take substring of first word with runes.
	// ... This handles any kind of rune in the string.
	runes := []rune(value)
	// ... Convert back into a string from rune slice.
	// 65 is max character length for role session name
	if len(runes) > 65 {
		safeSubstring = string(runes[0:65])
	} else {
		safeSubstring = string(runes[0:])
	}
	return safeSubstring
}

// contains returns true if slice contains string; false otherwise
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func checkDuration(duration time.Duration, min time.Duration, max time.Duration) error {
	if duration < min {
		return fmt.Errorf("Minimum assume-role duration is: (%s) duration requested: (%s) ", min.String(), duration)
	} else if duration > max {
		return fmt.Errorf("Maximum assume-role duration is: (%s) duration requested: (%s) ", max.String(), duration)
	}
	return nil
}
