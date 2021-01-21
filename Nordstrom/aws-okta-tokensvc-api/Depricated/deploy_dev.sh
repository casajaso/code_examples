#!/usr/bin/env bash
export AWS_DEFAULT_REGION=us-west-2
export TEAM_POLICY_ARN="arn:aws:iam::125237321655:policy/appteam/fff000-iam-nonprod-iepc-team01-TeamManagedPolicy-185PPIYZQFOU6"
export ENV_NAME="dev"
export DOMAIN_NAME="dev.cloud.vip.nordstrom.com"
export VERSION="v2"
export ACM_CERTIFICATE_ARN="arn:aws:acm:us-east-1:028824555316:certificate/278d72ab-7c83-4836-8f37-4452aeddd8ac"
export HOSTED_ZONE_ID="Z2K8R946NVT275"
export LDAP_LAMDA="cloud-api-ldap-dev-listCloudGroups"
echo "Deploying to Development AWS environment"
env GOOS=linux go build -ldflags="-s -w" -o bin/main
serverless deploy --stage dev