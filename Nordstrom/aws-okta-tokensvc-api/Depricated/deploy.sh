#!/bin/bash

if (( $# != 1 ))
then
  echo "Usage: Need to pass environment as argument (dev or prod)"
  exit 1
fi

ENV=$1

# Build Linux binary for AWS Lambda
env GOOS=linux go build -ldflags="-s -w" -o bin/main

if [[ $ENV = "prod" ]]; then

  echo "Deploying to Production AWS environment"

  export TEAM_POLICY_ARN="arn:aws:iam::028824555316:policy/appteam/fff000-iam-prod-iepc-team01-TeamManagedPolicy-A9UGTL4DY0OG"
  export ENV_NAME="prod"
  # export OKTA_ORGANIZATION="nordstrom"
  # export OKTA_AUTH_SERVER="okta.com/oauth2/aus28y44vpufBTqIG2p7"
  # export OKTA_OIDC_AUD="cli.aws.nordstrom.com"
  export DEFAULT_ROLE_SESSION_NAME="okta.session"
  export DOMAIN_NAME="tokenV2.cloud.vip.nordstrom.com"
  export ACM_CERTIFICATE_ARN="arn:aws:acm:us-east-1:028824555316:certificate/278d72ab-7c83-4836-8f37-4452aeddd8ac"
  export HOSTED_ZONE_ID="Z3UVDTD6UOEJDR"
  export LDAP_LAMDA="cloud-api-ldap-prod-listCloudGroups"

  sls create_domain --aws-profile nordstrom-federated --stage prod

  sls deploy --aws-profile nordstrom-federated --stage prod

elif [[ $ENV = "dev" ]]; then

  echo "Deploying to Development AWS environment"

  export TEAM_POLICY_ARN="arn:aws:iam::125237321655:policy/appteam/fff000-iam-nonprod-iepc-team01-TeamManagedPolicy-185PPIYZQFOU6"
  export ENV_NAME="dev"
  # export OKTA_ORGANIZATION="nordstrom"
  # export OKTA_AUTH_SERVER="okta.com/oauth2/aus28y44vpufBTqIG2p7"
  # export OKTA_OIDC_AUD="cli.aws.nordstrom.com"
  export DOMAIN_NAME="tokenV2.dev.cloud.vip.nordstrom.com"
  export DEFAULT_ROLE_SESSION_NAME="okta.session"
  export ACM_CERTIFICATE_ARN="arn:aws:acm:us-east-1:028824555316:certificate/278d72ab-7c83-4836-8f37-4452aeddd8ac"
  export HOSTED_ZONE_ID="Z3UVDTD6UOEJDR"
  export LDAP_LAMDA="cloud-api-ldap-dev-listCloudGroups"

  sls create_domain --aws-profile nordstrom-federated --stage dev

  sls deploy --aws-profile nordstrom-federated --stage dev

else

  echo "Invalid environment argument passed (dev or prod)"
  exit 1

fi
