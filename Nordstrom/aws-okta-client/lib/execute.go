/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

type Env []string

//ExecPerams defines
type ExecPerams struct {
	Profile     string
	Credentials credentials.Value
	Env         Env
	Interactive bool
}

func NewExecProvider(profile string, credentials credentials.Value, region string) *ExecPerams {
	e := GetEnviron(profile, region, &credentials)

	return &ExecPerams{
		Profile:     profile,
		Env:         *e,
		Credentials: credentials,
	}
}

//EnvVarHandler configures env vars for child exec process
func GetEnviron(profile string, region string, credentials *credentials.Value) *Env {
	e := Env(os.Environ())
	e.ClearVariables()
	e.Set("AWS_ACCESS_KEY_ID", credentials.AccessKeyID)
	e.Set("AWS_SECRET_ACCESS_KEY", credentials.SecretAccessKey)
	if credentials.SessionToken != "" {
		e.Set("AWS_SESSION_TOKEN", credentials.SessionToken)
		e.Set("AWS_SECURITY_TOKEN", credentials.SessionToken)
	}
	if region != "" {
		e.Set("AWS_DEFAULT_REGION", region)
		e.Set("AWS_REGION", region)
	}
	return &e
}

//ClearConflicts removes env vars in `ClearVars` array known to cause conflict
func (e *Env) ClearVariables() {
	for _, key := range ClearVars {
		e.Unset(key)
	}
}

//Unset removes environment variable
func (e *Env) Unset(key string) {
	for i := range *e {
		if strings.HasPrefix((*e)[i], key+"=") {
			(*e)[i] = (*e)[len(*e)-1]
			*e = (*e)[:len(*e)-1]
			break
		}
	}
}

//Set adds an environment variable, over-riding user defined vars in child shell
func (e *Env) Set(key, val string) {
	e.Unset(key)
	*e = append(*e, key+"="+val)
}

//GetExecPath get command path
func GetExecPath(command string) (string, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("Unable to locate (%v) in current dir or by environment variables", command)
	}
	return path, nil
}

//Contains searches string array for string
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// PowerShell structure
type PowerShell struct {
	powerShell string
}

// NewPSFromEnv determine ebs-path for PowerShell bin
func NewPSFromEnv() (*PowerShell, error) {
	if Platform == "windows" {
		path, err := exec.LookPath("powershell.exe")
		if err != nil {
			return nil, fmt.Errorf("Unable to locate powershell.exe - insure its installed and in user PATH")
		}
		return &PowerShell{
			powerShell: path,
		}, nil
	} else {
		return nil, fmt.Errorf("This option is a `Work in Progress` and currently only supported on Windows hosts")
	}
}

// Execute args in NewPSFromEnv
func (p *PowerShell) Execute(args []string) (stdOut string, stdErr string, err error) {
	args = append([]string{"-NoProfile", "-NonInteractive"}, args...)
	cmd := exec.Command(p.powerShell, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	stdOut, stdErr = stdout.String(), stderr.String()
	return
}
