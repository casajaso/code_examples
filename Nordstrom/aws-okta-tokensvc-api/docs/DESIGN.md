# Design Spec

We almost didn't have to implement a new Token Service. In order to meet MFA and compliance requirements, we can reevaluate our situation and potentially retire the Token Service if one of these 3 conditions are met:

- AWS provides support to evaluate Custom Claims for IdP tokens inside an IAM Role Trust Relationship (preferred)
- Global MFA is enabled across the Nordstrom Organization
- Okta makes available an app-level MFA API

Please note that any mention of a groups claim in this documentation is **DEPRECATED** functionality, but has been left in this document for legacy purposes.

[Okta](https://www.okta.com/) does not offer an app-level **Multi-Factor Authentication (MFA)** API. This means that if global MFA is not set across the organization, the MFA API will not work for app-level MFA enablement. As an organization, we are not in a position yet to enable MFA globally. We also cannot use AWS for MFA. Federated users cannot be assigned an MFA device for use with AWS services, so they cannot access AWS resources controlled by MFA. More info on AWS restrictions [here](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_mfa_configure-api-require.html). In retrospect, this makes sense as AWS doesn't have a means to store MFA device information per federated user.

Because of these limitations, the `aws-okta` CLI is configured to authenticate against a native Okta OIDC application as opposed to the more traditional Okta SAML application. The industry, however, is making a huge shift to use OIDC applications for command line utilities (e.g., the Google `gcloud` CLI and Azure CLI). The good news is that this approach allows us to still use Okta as an Identity Provider.

The Okta OIDC application is configured to perform *Auth Code Flow with PKCE*. You can learn more about *Auth Code Flow with PKCE* [here](https://developer.okta.com/authentication-guide/implementing-authentication/auth-code-pkce). This configuration requires the user to login into a browser for an Okta session. Because it is via an Okta-controlled web page, the user's Okta credentials are never passed through the `aws-okta` CLI.

The new **AWS Okta Token Service** assumes IAM Roles on behalf of the user. The *Request Body* is identical to the AWS STS `AssumeRole` operation as is the *Response Body*. In essence, it serves as a "proxy". The **AWS Okta Token Service** validates the IdP token passed in the `Authorization` Header and determines what IAM roles the user can assume. If the user is authorized to assume the IAM Role passed as a parameter, the **AWS Okta Token Service** will assume the IAM Role on behalf of the user. It will prefix the Role Session Name with the `sub` claim derived from the IdP token, which is Okta's unique identifier for the authenticated user. If the Role Session Name is left blank, a default Role Session Name will be used.

## AWS IAM TRUST POLICY

At this time, AWS does **NOT** support evaluating custom claims of an IdP token when using an OIDC Federated Identity Provider. This means there is no way to restrict CLI users from accessing AWS accounts they should not be able to access when configuring the IAM Role's trust relationship.

If you just create a Trust Policy for your IAM Role that only includes the Client ID of the Okta OIDC application, then any team who is whitelisted to use the CLI would be able to assume the AWS IAM Role.

For example, it's not secure to only add the OIDC Client ID used for authentication in the IAM condition. In this example, we configure a Trust Policy for IAM Role, `NORD-SANDBOXCORE-DevUsers-Team`, and AWS Account, `355918624725`. The OIDC Client ID is `0oagdckynkMFS57ZY0h7`. But anyone who is whitelisted to use the CLI can access this role:

```json
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::355918624725:oidc-provider/nordstrom.okta.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "nordstrom.okta.com:aud": "0oagdckynkMFS57ZY0h7"
        }
      }
    }
  ]
}
```

The IdP token contains a Custom Claim called `groups` that is an array of all the AWS Security Groups associated to the user. Because of this, we should be able to write another IAM condition to evaluate if the `groups` Custom Claim contains a string value of a specified AWS AD Security Group. This would allow us to lock down who is able to assume the IAM Role. It'd look something like this:

```json
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::355918624725:oidc-provider/nordstrom.okta.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "nordstrom.okta.com:aud": "0oagdckynkMFS57ZY0h7"
        },
        "ForAnyValue:StringEquals": {
          "nordstrom.okta.com:groups": "AWS#NORD-SANDBOXCORE-DevUsers-Team#355918624725"
        }
      }
    }
  ]
}
```

However, this doesn't work! IAM conditions are only able to evaluate Standard Claims and not Custom Claims. `groups` is a Custom Claim. Under the hood, IAM converts the Standard Claims to IAM conditional keys. `nordstrom.okta.com:aud` is an example of an IAM converted key. At the time of this writing, IAM does not do this for Custom Claims on the IdP token. Nordstrom has requested support for this functionality.

The **AWS Okta Token Service** will be able to go away upon release of Custom Claim evaluation in IAM Role Trust Policies for OIDC federated Identity Providers or possibly when Okta supports an app-level MFA API.

In the interim, we add this policy to each IAM Role's trust relationship for the **AWS Okta Token Service** to assume a role on the user's behalf:

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

Upon the retirement of the **AWS Okta Token Service**, the `aws-okta` CLI will be able to perform the `AssumeRoleWithWebIdentity` operation directly.

## Auth Code Flow with PKCE

The Okta native OIDC application is configured for _Authorization Code Flow with a Proof Key for Code Exchange (PKCE)_.

The _Authorization Code Flow with PKCE_ is the standard Code flow with an extra step at the beginning and an extra verification at the end. At a high-level, the flow has the following steps:

* The CLI generates a code verifier followed by a code challenge.
* The CLI then directs the browser to the Nordstrom Okta Sign-In page, along with the generated code challenge, and the user authenticates.
* Okta redirects back to the CLI native application with an authorization code. The CLI spins up a temporary server for this action to be done.
* The CLI application sends this code, along with the code verifier, to Okta. Okta returns access and ID tokens, and optionally a refresh token.
* The CLI can now use these tokens to call the Token Service to assume an IAM Role on their behafl..

For more information on [Auth Code Flow with PKCE](https://developer.okta.com/authentication-guide/implementing-authentication/auth-code-pkce).

## AWS Resources

- [Creating OpenID Connect (OIDC) Identity Providers](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html)
- [Prerequisites for Creating a Role for Web Identity or OIDC](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp_oidc.html#idp_oidc_Prerequisites)

## Okta Resources

- [OpenID Connect & OAuth 2.0 API](https://developer.okta.com/docs/api/resources/oidc)
