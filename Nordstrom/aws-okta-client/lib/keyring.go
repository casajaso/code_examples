/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/99designs/keyring"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	passwd string
)

func keyringPrompt(prompt string) (string, error) {
	return PromptWithOutput(prompt, true, os.Stderr)
}

// Prompt calls PromptWithOutput for unlocking the keychain
func Prompt(prompt string, sensitive bool) (string, error) {
	return PromptWithOutput(prompt, sensitive, os.Stdout)
}

// Prompt retrieves keychain credentials from user
func PromptWithOutput(prompt string, sensitive bool, output *os.File) (string, error) {
	fmt.Fprintln(os.Stdout, "%s: ", prompt)
	defer fmt.Fprintf(output, "\n")
	if sensitive {
		var input []byte
		input, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(string(input)) == "" {
			err := fmt.Errorf("Password cannot be blank")
			return "", err
		}
		return strings.TrimSpace(string(input)), nil
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		err := fmt.Errorf("Password cannot be blank")
		return "", err
	}
	return strings.TrimSpace(value), nil
}

// OpenKeyring unlock keyring to store / retieve keyring items
func NewKeyringSession(backends []keyring.BackendType) (kr keyring.Keyring, err error) {
	kr, err = keyring.Open(keyring.Config{
		AllowedBackends:          backends,
		KeychainTrustApplication: true,
		ServiceName:              "aws-okta",
		LibSecretCollectionName:  "aws-okta",
		FileDir:                  "~/.aws-okta/",
		KeychainPasswordFunc:     keyringPrompt,
		FilePasswordFunc:         keyringPrompt,
	})

	return
}
