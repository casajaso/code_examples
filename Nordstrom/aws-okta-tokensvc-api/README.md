# AWS Okta Token Service

The **AWS Okta Token Service** assumes an IAM Role on behalf of an authenticated Okta user and returns back AWS temporary security credentials.

[![pipeline status](https://gitlab.nordstrom.com/public-cloud/aws-okta-token-service/badges/master/pipeline.svg)](https://gitlab.nordstrom.com/public-cloud/aws-okta-token-service/commits/master)

**Process Flow:**

1. After user is authenticated with the Okta AWS OIDC application via the [AWS Okta CLI](https://gitlab.nordstrom.com/public-cloud/aws-okta), Okta sends an IdP token.

2. Upon receiving a valid request to assume an IAM Role on behalf of the user, the **Okta Token Service** evaulates if the IdP token provided via the Authorization Headers is valid.

3. Next, the **Okta Token Service** sees if the user is authorized to assume the IAM Role requested. It performs this action by hitting a lambda to see what groups the user is a member of.

4. If the user is authorized to assume the IAM Role, the **Okta Token Service** returns temporary security credentials to the client.

To learn more about the [design](docs/DESIGN.md).

## Environmental Information

**Endpoints:**

| Environment | Endpoint                                       |
| :---------: | :--------------------------------------------: | 
| Prod        | https://tokensvc.prod.cloud.vip.nordstrom.com  |
| NonProd     | https://tokensvc.dev.cloud.vip.nordstrom.com   |

**Okta:**

| Environment | Okta Auth Server                                              | Okta OIDC App ID     |
| :---------: | :-----------------------------------------------------------: | :------------------: |
| Prod        | https://nordstrom.okta.com/oauth2/aus28y44vpufBTqIG2p7        | 0oa28xw7eaSoJF6NK2p7 |
| NonProd     | https://nordstrom.oktapreview.com/oauth2/ausgey4fpt3X8zDbq0h7 | 0oagdckynkMFS57ZY0h7 |

## Routes

The **AWS Okta Token Service** only exposes one endpoint, [POST] `/`.

### Request Body

| Key             | Type   | Required | Example                                                           |
| :-------------: | :----: | :------: | :---------------------------------------------------------------: |
| RoleArn         | String | Yes      | "arn:aws:iam::468543742856:role/NORD-SANDBOXTEAM01-DevUsers-Team" |
| RoleSessionName | String | Yes      | "MyRoleSessionName"                                               |
| DurationSeconds | Number | Yes      | 900                                                               |
| Policy          | String | No       | {"Version":"2012-10-17","Statement":[{"Sid":"DenyAllExceptS3","Effect":"Deny","NotAction":["s3:*"],"Resource":"*"},{"Sid":"AllowS3","Effect":"Allow","Action":["s3:*"],"Resource":"arn:aws:s3:::*"}]} |

> **NOTE:** **Duration Seconds** can be from *900 seconds (15 minutes)* up to the maximum session duration setting for the role. If not a production PCI account, IAM roles are configured for a maximum duration of *43200 seconds (12 hours)*. **RoleSessionName** is an identifier for the assumed role session. The string of characters can consist of upper- and lower-case alphanumeric characters with no spaces. You can also include underscores or any of the following characters: `=,.@-`. If **RoleSessionName** is left blank, the **AWS Okta Token Service** uses a default Role Session Name. It'll also prefix the Role Session Name with the Okta's unique identifier for the authenticated user. If length of the string becomes longer than the maximum 65 character limit, the string will be truncated.

**Request Body Example:**

```json
{
  "RoleArn": "arn:aws:iam::468543742856:role/NORD-SANDBOXTEAM01-DevUsers-Team",
  "RoleSessionName": "MyRoleSessionName",
  "DurationSeconds": 900,
  "Policy": '{"Version":"2012-10-17","Statement":[{"Sid":"DenyAllExceptS3","Effect":"Deny","NotAction":["s3:*"],"Resource":"*"},{"Sid":"AllowS3","Effect":"Allow","Action":["s3:*"],"Resource":"arn:aws:s3:::*"}]}'
}
```

### Request Headers

| Key             | Type   | Required | Example                   |
| :-------------: | :----: | :------: | :-----------------------: |
| Authorization   | String | Yes      | "eyJraWQiOiJlLUhGNFhWQ.." |
| Content-Type    | String | Yes      | "application/json"        |

**Request Headers Example:**

```json
{
  "Authorization": "eyJraWQiOiJlLUhGNFhWQ...",
  "Content-Type": "application/json"
}
```

### Response

**Response Example:**

```json
{
    "Credentials": {
        "AccessKeyId": "REDACTED",
        "SecretAccessKey": "REDACTED",
        "SessionToken": "REDACTED",
        "Expiration": "2018-10-05T22:34:22Z"
    },
    "AssumedRoleUser": {
        "AssumedRoleId": "AROAIGVVDF446J6GVECTA:my-test-session-1000",
        "Arn": "arn:aws:sts::468543742856:assumed-role/NORD-SANDBOXTEAM01-DevUsers-Team/my-test-session-1000"
    }
}
```

### IdP Token

**Example of decoded Okta ID Token:**

```json
{
  "sub": "00umqxus8TROl4PPS2p6",
  "ver": 1,
  "iss": "https://nordstrom.okta.com/oauth2/aus28y44vpufBTqIG2p7",
  "aud": "0oa28xw7eaSoJF6NK2p7",
  "iat": 1538776683,
  "exp": 1538780283,
  "jti": "ID.QWSjBqIEHVo4VJchhojtMZ2jVkqSYMPixZg54kP5qNM",
  "amr": [
    "pwd"
  ],
  "idp": "00o7ktbrffAjF2gGY2p6",
  "nonce": "nonce",
  "auth_time": 1000,
  "at_hash": "preview_at_hash",
  "groups": [
    "AWS#NORD-SANDBOXTEAM01-CloudPlatform-Team#468543742856",
    "AWS#NORD-SANDBOXTEAM01-DevUsers-Team#468543742856",
    "AWS#NORD-SANDBOXTEAM02-DevUsers-Team#043457593234"
  ]
}
```

## CI/CD
### Testing

To execute unit tests:

```bash
# go test will run unit tests
$ go test
# PASS
# ok  	gitlab.nordstrom.com/public-cloud/aws-okta-token-service	0.009s

# go test coverage will analyze coverage
$ go test -cover
# PASS
# coverage: 21.5% of statements
# ok  	gitlab.nordstrom.com/public-cloud/aws-okta-token-service	0.010s
```
### Pipeline Deployment
To deploy with the pipeline, commit changes to this repo. Any changes to the branch `integration`
will push the code a S3 bucket in the NonProd IEPC account. In addition, it will do a CFT deploy
allowing for the changes to be visible in the development stage.

Pushing changes to `master` will push to the prod environment. Any changes to the master brnach will have
the code pushed to a S3 bucket in Prod IEPC. Similarly to the dev environment, the pipeline with deploy
the CFT allowing changes to be seen in the production environment.

### Manual Deployment with the CLI
For manual deployments, follow the steps below:
1. Use aws-okta to authenticate to the account/environment you wish to deploy in.
  > Use NonProd_IEPC_DevUser to deploy to the Dev environment. Use Prod_IEPC_DevUser to deploy to the
  production environment. NEVER use a CloudEng role as the pipeline will have issues in the future if
  resources are deployed with CloudEng that are outside the DevUser permissions.

2. Get the name of the stack from the account you are deploying to.
3. Compile the code
```bash
go build main.go
```
4. Zip the executable file with a unique name.
5. Copy the zip file to the S3 bucket in the account you are deploying to.
  > The bucket name is aws-okta-token-service-<env>, where <env> is dev or prod.

6. Set up the environment variables the stack needs.
  > Use the commands below with the correct environment set (nonprod or prod)

  ```bash
  environment=$(cat config.json | jq -r '.account_information .<env> .env')
  cert=$(cat config.json | jq -r '.account_information .<env> .acm_cert_arn')
  ```
7. Run the CFT deploy command.
> Make sure to replace <Key> in the command below with the name of the zip in the S3 bucket from step
5

```bash
  aws cloudformation deploy --template ./cloudformation_templates/tokensrv-cft.yaml --stack-name aws-okta-tokensvc --parameter-overrides Environment=${environment} ACMCert=${cert} LambdaCodeKey=<Key>.zip --capabilities CAPABILITY_NAMED_IAM
```

8. Ensure that the stack deploys properly.

## Integration with IAM Role to assume

The trust relationship of the IAM Role to assume needs to have this trust policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "okta-token-service",
      "Effect": "Allow",
      "Principal": { "AWS": "arn:aws:iam::028824555316:role/fff000-Okta-Token-Service" },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

This trust policy will allow the **AWS Okta Token Service** to assume the requested IAM Role on behalf of the authenticated user. Moreover, the AWS Security Group associated to this AWS IAM Role will need to be added to the AWS Okta OIDC's whitelist of AWS AD Security Groups. The format of the AD Security Group needs to be in this format, `AWS#<IAM_ROLE_NAME>#<AWS_ACCOUNT_NUMBER>` (e.g., `AWS#NORD-SANDBOXTEAM01-CloudPlatform-Team#468543742856`)

## Further Reading

* [Okta OIDC with Auth Code Flow with PKCE](https://developer.okta.com/authentication-guide/implementing-authentication/auth-code-pkce)
* [IAM AssumeRole](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
