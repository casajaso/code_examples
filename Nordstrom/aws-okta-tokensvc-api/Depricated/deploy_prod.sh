#!/usr/bin/env bash
export AWS_DEFAULT_REGION=us-west-2
export TEAM_POLICY_ARN="arn:aws:iam::028824555316:policy/appteam/fff000-iam-prod-iepc-team01-TeamManagedPolicy-A9UGTL4DY0OG"
export ENV_NAME="prod"
export DOMAIN_NAME="cloud.vip.nordstrom.com"
export VERSION="v2"
export ACM_CERTIFICATE_ARN="arn:aws:acm:us-east-1:028824555316:certificate/2b85c9b9-fc37-4dbc-9782-c2fcded761a1"
export HOSTED_ZONE_ID="Z2FDTNDATAQYW2"
export LDAP_LAMDA="cloud-api-ldap-prod-listCloudGroups"
echo "Deploying to Production AWS environment"
env GOOS=linux go build -ldflags="-s -w" -o bin/main
serverless deploy --stage prod