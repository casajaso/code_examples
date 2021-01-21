# aws-okta

**aws-okta** is a CLI utility that allows Nordstrom employees to authenticate with AWS using your Okta credentials.

[![pipeline status](https://gitlab.nordstrom.com/public-cloud/aws-okta/badges/master/pipeline.svg)](https://gitlab.nordstrom.com/public-cloud/aws-okta/commits/master)

Inspired by the Segmentio [project](https://github.com/segmentio/aws-okta), it has been modified to adhere to Nordstrom security requirements and enterprise needs, including PCI compliance and Multi-Factor Authentication (MFA). The project relies on the [aws-vault](https://github.com/99designs/aws-vault) library to securely store and access credentials for AWS. It does this by storing your AWS credentials in your operating system's secure keystore and then generates temporary credentials from those to expose to your shell and applications. It's designed to be complementary to the AWS CLI tools, and is aware of your profiles and configuration in `~/.aws/config`. It does not use `~/.aws/credentials` to store AWS security credentials in plain text.

When assuming an IAM Role, **aws-okta** makes a call to the [AWS Okta Token Service](https://gitlab.nordstrom.com/public-cloud/aws-okta-token-service) to retrieve temporary AWS security credentials. For more information, please see the [Design Spec](https://gitlab.nordstrom.com/public-cloud/aws-okta-token-service/blob/master/docs/DESIGN.md).

## How to install

The **aws-okta** CLI binary is available for Linux, Windows, and MacOS O/S distrobutions. You can find the latest stable release summery.  <a href="https://storage.googleapis.com/aws-okta/release/stable.txt">here</a>

<details>
<summary>MacOS</summary>

<br/>
Download the latest release and make executable:
<br/>
<pre>
$ curl -o /usr/local/bin/aws-okta "https://storage.googleapis.com/aws-okta/release/LATEST/bin/darwin/x64/aws-okta"
$ chmod +x /usr/local/bin/aws-okta
</pre>

</details>

<details>
<summary>Linux</summary>
<strong>
Linux Binary is in early development and not currently supported
</strong>
<br/>
<pre>
$ curl -sLo /usr/local/bin/aws-okta "https://storage.googleapis.com/aws-okta/release/LATEST/bin/linux/x64/aws-okta"
$ chmod +x /usr/local/bin/aws-okta
</pre>

</details>

<details>
<summary>Windows</summary>

<br/>
* Download the latest release <a href="https://storage.googleapis.com/aws-okta/release/LATEST/bin/win/x64/aws-okta.exe">here</a>
<br/>
Add the binary in to your <strong>PATH</strong>.

</details>

## Prequisites

# Minimum Requirments
1. [AWS Command Line Interface](https://docs.aws.amazon.com/cli/latest/userguide/installing.html) - With Boto3: 1.8.0 or higher
    ```pip install aws-cli
    pip install boto3```

# Developer Requirments: AWS-SDK
  - Python (boto3): 1.8.0
  - Node: v1.16.0 (2018-12-05)
  - GoLang: v1.16.0 (2018-12-05)
  - Java: 1.11.489 2019-01-24
  For other laungauges check AWS-SDK repo/CHANGELOG and search for `credential process`, `credential_process` or `process_credentials`


## Usage
`aws-okta` or `aws-okta help` dispays a brief discription for each top level command.
`aws-okta help <command-name>` displays detailed infomation on each top level command.


## Order of Precedence

The order of precedence for user-definables: highest to lowest

1. Command Line Flags
2. Environmental Variables
3. AWS Config Profile Attributes

### Login Command

```bash
$ aws-okta login -p (pci) Omitting defaults to (non-pci)
```

Login will open up the **Nordstrom** Single Sign-ON (SSO) page in a browser window. If the user does not have a un-expired Okta session for the PCI or Non-PCI OIDC endpoint they will be prompted with an MFA challenge. Upon successful authentication, the **aws-okta** will retrieve the IdP Token from the Okta OIDC application. This process utilizes the *Auth Code Flow with PKCE* to retrieve tokens . Non-PCI tokens expire in <= 12 hours and PCI tokens expire hourly. Upon expiration, the user will be prompted with another MFA challenge.

```bash
login will authenticate you through okta and allow you to access your AWS environments

Usage:
  aws-okta login [flags]

Flags:
  -d, --dev    Direct login to use dev endpoints (unset defaults to prod)
  -h, --help   help for login
  -p, --pci    Direct login to use pci endpoints (unset defaults to non-pci)

Global Flags:
  -b, --backend string     Specify which backend credential store to use [keychain pass file]
  -v, --verbosity string   Sets verbocity/log-level Options: [WARN, INFO, DEBUG] (default: WARN) (default "warn")
```

> **NOTE:** When prompted with MFA challenge you'll be directed to your browser to complete the login challenge. Unfortunately Okta APIs do not directly support app-level MFA, which is why opening the browser is required.
List will prompt the user to select an IAM Role from a dropdown of IAM Roles the user is allowed to assume. Upon the selection of an IAM Role, the IAM Role ARN chosen will be saved in the AWS Config file under the user defined profile or if ommited `nordstrom-federated`.

### Creds/List Command
```bash
Displays federated roles and updates aws/config with selected role

Usage:
  aws-okta creds <profile> [flags]

Aliases:
  creds, list

Flags:
  -t, --assume-role-ttl duration                  Expiration time for assumed role (default 1h0m0s)
  -h, --help                                      help for creds
  -p, --profile-name string                       Specify profile name (default "nordstrom-federated")
  -r, --role-arn (Bypass role selection prompt)   Specify sts-assume role_arn (Bypass role selection prompt)

Global Flags:
  -b, --backend string     Specify which backend credential store to use [keychain pass file]
  -v, --verbosity string   Sets verbocity/log-level Options: [WARN, INFO, DEBUG] (default: WARN) (default "warn")
```

```bash
example

$ aws-okta creds or list

Use the arrow keys to navigate: ↓ ↑ → ← (MacOS, Linux) or h j k l (Windows)
? Select an AWS Role:
  ▸ arn:aws:iam::468543742856:role/NORD-SANDBOXTEAM01-CloudPlatform-Team
    arn:aws:iam::468543742856:role/NORD-SANDBOXTEAM01-DevUsers-Team
    arn:aws:iam::043457593234:role/NORD-SANDBOXTEAM02-DevUsers-Team
```

 A profile section is created/updated in `~/.aws/config` and `~/.aws/credentials` enabling `aws credential_process` providing access to authenticated `aws cli` for defined profilename to selected role.

If the specified `profile-name`  is already exists in `~/.aws/config` or `~/.aws/credentials` the previous profile section is cleared and a new section is written.

```ini
[profile nordstrom-federated]
_role_arn          = arn:aws:iam::<some-nonpci-accountid>:role/<some-non-pci-role>
credential_process = aws-okta-dev cproc --profile-name nordstrom-federated --assume-role-ttl 1hr0m0s
[profile nordstrom-federated-pci]
_role_arn          = arn:aws:iam::<some-pci-accountid>:role/<some-pci-role>
credential_process = aws-okta-dev cproc --profile-name nordstrom-federated-pci --assume-role-ttl 15m0s
```

```bash
$ aws sts get-caller-identity --profile nordstrom-federated
{
    "UserId": "<firstname>.<lastname>@nordstrom.com-okta.session",
    "Account": "<some-accounid>",
    "Arn": "arn:aws:sts::<some-accounid>:role/<some-role-name>/<firstname>.<lastname>@nordstrom.com-okta.session"
}
```
### Export Command
```bash
export will export credentials to stdout or .net sdk for AWS powershell cmdlett

Usage:
  aws-okta export <profile> [flags]

Flags:
  -t, --assume-role-ttl duration   Expiration time for assumed role (default 1h0m0s)
  -e, --export-to string           Specify export target [STDOUT, POWERSHELL] (default "stdout")
  -h, --help                       help for export
  -p, --profile-name string        Specify profile name (default "nordstrom-federated")

Global Flags:
  -b, --backend string     Specify which backend credential store to use [keychain pass file]
  -v, --verbosity string   Sets verbocity/log-level Options: [WARN, INFO, DEBUG] (default: WARN) (default "warn")
```
This output can be set to Environment variables for use with other applications by:

MacOS:
```bash
$ eval $(aws-okta export -e stdout nordstrom-federated | sed 's/^/export /')
```
Windows:
```powershell
PS: aws-okta.exe creds -o -p nordstrom-federated | %{$_ -replace "^",'$env:'} | ForEach-Object {Invoke-Expression $_}
```
Storing Credentials for `Windows PowerShell for AWS`:
> **NOTE**: This process writes the CURRENT aws session token to the .NET credential store (valid for up to 1hr) and needs to be ran periodicly in order to refresh credentialdata
```powershell
PS: aws-okta export -e powershell -p nordstrom-federated
```
now when you open `Windows PowerShell for AWS` you should see an entry for the profile `nordstrom-federated`
```powershell
Saved credentials found
Please choose one of the following stored credentials to use
[] <Create new credentials set>  [] nordstrom-federated  [?] Help (default is "<Create new credentials set>"): nordstrom-federated
PS C:\Program Files (x86)\AWS Tools\PowerShell\AWSPowerShell>Get-STSCallerIdentity

Account    Arn                                                                                        UserId
-------    ---                                                                                        -----
0123456789 arn:aws:sts::0123456789:assumed-role/SANDBOXTEAM02_DevUsers/*.***@nordstrom.com-okta.session A*...
```

### Exec Command

```bash
exec will run the command specified in an authenticated child-shell

Usage:
  aws-okta exec <profile> -- <command> [flags]

Flags:
  -t, --assume-role-ttl duration   Expiration time for assumed role (default 1h0m0s)
  -h, --help                       help for exec
  -p, --profile-name string        Specify profile name (default "nordstrom-federated")

Global Flags:
  -b, --backend string     Specify which backend credential store to use [keychain pass file]
  -v, --verbosity string   Sets verbocity/log-level Options: [WARN, INFO, DEBUG] (default: WARN) (default "warn")

```

### Refresh Command

```bash
refresh will expire and refresh AWS session token

Usage:
  aws-okta refresh <profile> [flags]

Flags:
  -t, --assume-role-ttl duration   Expiration time for assumed role (default 1h0m0s)
  -h, --help                       help for refresh
  -p, --profile-name string        Specify profile name (default "nordstrom-federated")

Global Flags:
  -b, --backend string     Specify which backend credential store to use [keychain pass file]
  -v, --verbosity string   Sets verbocity/log-level Options: [WARN, INFO, DEBUG] (default: WARN) (default "warn")
```

Refresh will attempt to force expire current AWS session token, remove session token from keyring and request a new one.


### Logout Command

```bash
logout will revoke aws credentials and terminate session

Usage:
  aws-okta logout <profile> [flags]

Flags:
  -h, --help   help for logout

Global Flags:
  -b, --backend string     Specify which backend credential store to use [keychain pass file]
  -v, --verbosity string   Sets verbocity/log-level Options: [WARN, INFO, DEBUG] (default: WARN) (default "warn")
```

Logout will attempt to force expire current AWS session token and remove all AWS session token from the keyring as well as the curent Okta session token.


### Console Command

```bash
$ aws-okta console <profile>
```

Console will allow you to access your AWS environment through a browser. You first need to be authenticated with Okta (`aws-okta login`). The Console command accepts the following command line flags:

```bash
$ aws-okta help console
console will allow you to access your AWS environment through a browser

Usage:
  aws-okta console <profile> [flags]

Flags:
  -t, --assume-role-ttl duration   Expiration time for assumed role (default 1h0m0s)
  -h, --help                       help for console
  -p, --profile-name string        Specify profile name (default "nordstrom-federated")
  -s, --stdout                     Print login URL instead of attempting to open default browser

Global Flags:
  -b, --backend string     Specify which backend credential store to use [keychain pass file]
  -v, --verbosity string   Sets verbocity/log-level Options: [WARN, INFO, DEBUG] (default: WARN) (default "warn")
```

## Backends

**aws-okta** use *99design's* keyring package that is also used in **aws-vault**.  Because of this, you can choose between different pluggable secret storage backends just like in **aws-vault**. You can either set your backend from the command line as a flag, or set the `AWS_OKTA_BACKEND` environment variable; However note that you are limited dicoverable backends for your operating system.

For Linux / Ubuntu add the following to your bash config / [zshrc](https://github.com/robbyrussell/oh-my-zsh) etc:

```
export AWS_OKTA_BACKEND=secret-service
```

## Environmental Variables

### User

| Environment Variable     | Description                                                 | Example                                                         |
| :----------------------: | :---------------------------------------------------------: | :-------------------------------------------------------------: |
| AWS_ASSUME_ROLE_TTL      | The length to assume the IAM role                           | 5h15m0s                                                |
| AWS_CONFIG_FILE          | Override the default ~/.aws/config file                     | ~/.aws/config                                                   |
| AWS_OKTA_BACKEND         | Choose different pluggable secret storage backends          | secret-service                                                  |
| AWS_OKTA_IAM_ROLE        | Provide an IAM Role ARN to avoid Prompt when using `list`   | arn:aws:iam::468543742856:role/NORD-SANDBOXTEAM01-DevUsers-Team |
| AWS_OKTA_DEFAULT_PROFILE | Override Profile to save selected IAM Role ARN using `list` | dev-okta                                

### Okta

Overriding these configurations would most likely be for development on the AWS Okta CLI.

| Environment Variable          | Description                    |  Example                                                         |
| :---------------------------: | :----------------------------: | :--------------------------------------------------------------: |
| AWS_OKTA_ORGANIZATION         | Okta Organization              | "nordstrom"                                                      |
| AWS_OKTA_SERVER               | Okta Auth Server               | "okta.com/oauth2/aus28y44vpufBTqIG2p7"                           |
| AWS_OKTA_KEY_SESSION          | Okta Keyring Session Name      | "OktaAuthSession"                                                |
| AWS_OKTA_REDIRECT_ADDR        | Okta Redirect Host             | "localhost:5556"                                                 |
| AWS_OKTA_REDIRECT_PATH        | Okta Redirect URL Path         | "/auth/okta/callback"                                            |
| AWS_OKTA_SUCCESS_PAGE         | Okta Auth Success Landing Page | "https://s3-us-west-2.amazonaws.com/aws-auth-success/index.html" |
| AWS_OKTA_DEFAULT_SESSION_NAME | Okta Default Role Session Name | "okta.session"                                                   |
| AWS_OKTA_OIDC_APP_ID_KEY      | Okta OIDC Required Claim       | "oidc_app_id"                                                    |
| AWS_OKTA_OIDC_APP_ID          | Okta OIDC ID                   | "0oa28xw7eaSoJF6NK2p7"                                           |
| AWS_OKTA_TOKEN_SERVICE        | AWS Okta Token Service Host    | "https://token.cloud.vip.nordstrom.com"                          |


When a user *execs* a profile, that process will set the following environmental variables:

| Environment Variable  |
| :-------------------: |
| AWS_DEFAULT_REGION    |
| AWS_REGION            |
| AWS_ACCESS_KEY_ID     |
| AWS_SECRET_ACCESS_KEY |
| AWS_SESSION_TOKEN     |
| AWS_SECURITY_TOKEN    |
| AWS_OKTA_PROFILE      |

## Contributing

post an issue with your recommendation or submit a merge request.

### Installing

If you have a functional go environment, you can install with:

```bash
$ go get gitlab.nordstrom.com/public-cloud/aws-okta
```

When making a PR, please include unit tests and make sure coverage passes. For more information, please review our [Contribution Guidelines](https://gitlab.nordstrom.com/public-cloud/aws-okta/blob/master/CONTRIBUTING.md).

### Building Dev Binary

To build a local dev binary (targets dev okta end-points):

```bash
$ make development -e VERSION="<some-version-number>-dev" RELEASE="NORDSTROM AWS-OKTA CLI (DEVELOPMENT BUILD) - COMPILED ($(date +'%m/%d/%Y')) -$(whoami | tr a-z A-Z)"
```

### Tests

Go has a built-in testing command called `go test` and a package `testing`.

Unit tests for this project are stored in the directory, `test`.

To execute them, run this command:

```bash
$ go test -v ./tests
```

## Support

Please use the [#aws-okta](slack://channel?id=CD9NQJCMT&team=T02BEGF00) Slack channel for questions, feedback, and help.

If you come across an `Internal Server Error` message or similar, try again with the `--debug` (or `-d`) flag. This will print a Request ID that the Public Cloud team can use to help troubleshoot your issue.

```bash
$ aws-okta exec <profile> -- <command>
DEBU[0004] Request ID: f04b6648-cd49-11e8-94a6-5f6f5d45e8e7
```
