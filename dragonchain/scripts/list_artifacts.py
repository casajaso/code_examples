#!/usr/bin/env python
# 09/2018
# Itterates through checklist of baseline AWS resources needed to provision customer chain stacks

import os
import sys
import boto3
import argparse
import json
from numpy import size

def validate_resources(region, profile):
    session = boto3.Session(profile_name=profile)
    ec2_client = session.client('ec2', region_name=region)
    sts_client = session.client('sts', region_name=region)
    iam_client = session.client('iam', region_name=region)
    account_id = sts_client.get_caller_identity()["Account"]

    vpcs = ec2_client.describe_vpcs()['Vpcs']
    subnets = ec2_client.describe_subnets()['Subnets']
    security_groups = ec2_client.describe_security_groups()['SecurityGroups']
    internet_gateways = ec2_client.describe_internet_gateways()['InternetGateways']
    nat_gateways = ec2_client.describe_nat_gateways()['NatGateways']
    route_tables = ec2_client.describe_route_tables()['RouteTables']
    roles = [iam_client.get_role(RoleName='DRGNCoreLambdaRole')['Role']]
    policies = [iam_client.get_policy(PolicyArn='arn:aws:iam::{}:policy/DRGNCoreLambdaPolicy'.format(account_id))['Policy']]
    key_pairs = ec2_client.describe_key_pairs()['KeyPairs']

    validation_terms = {'vpcs' : vpcs, 'subnets' : subnets, 'security_groups' : security_groups, 'internet_gateways' : internet_gateways, 'nat_gateways' : nat_gateways, 'route_tables' : route_tables, 'roles' : roles, 'policies' : policies, 'key_pairs' : key_pairs}

    for term in validation_terms:
        print '{}'.format(term).upper()

        try:
            for resource in validation_terms[term]:
                for key1 in resource:
                    if size(resource[key1]) > 1:
                        for key2 in resource[key1]: 
                            print '\t {}: {}'.format(key1, key2)
                    else:
                        print '\t {}: {}'.format(key1, resource[key1])
                print ''
            print ''
        except Exception as ex:
                print('[ERROR][{}] {}'.format(term, ex))

def main(region, profile):
    validate_resources(region, profile)

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='List details for AWS environment artifacts')
    parser.add_argument('-r', '--region', nargs='?', default='', dest='region', help=' <ec2_region>')
    parser.add_argument('-p', '--profile', nargs='?', default='', dest='profile', help=' <aws_profile>')
    args = parser.parse_args()
    region = args.region
    profile = args.profile
    main(region, profile)
