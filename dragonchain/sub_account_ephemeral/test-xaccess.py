#!/usr/bin/python
# 08/2018
# validates artifacts created by bootstrap.py

import boto3
import botocore.config
import json
import argparse
import sys
from numpy import size

def get_current_id(session, boto_config): #USE THIS TO FIND OUT WHO YOU ARE WHEN DEBUGGING
    sts_client = session.client('sts', config=boto_config)
    response = sts_client.get_caller_identity()
    response.pop('ResponseMetadata')
    return response

def get_credentials(region, session, boto_config, credentials, assume_creds=''):
    if assume_creds == '':
        sts_client = session.client('sts', config=boto_config)
    else:
        access_key_id = assume_creds['AccessKeyId']
        secret_access_key =  assume_creds['SecretAccessKey']
        session_token = assume_creds['SessionToken']
        session = boto3.Session (aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
        sts_client = session.client('sts', config=boto_config)
        print '\tUsing Assumed Caller_id: {}'.format(get_current_id(session, boto_config))
    print '\tUsing Caller_id: {}'.format(get_current_id(session, boto_config))
    response = sts_client.assume_role(RoleArn=credentials, RoleSessionName='assumed-role-session')['Credentials']
    return response

def get_secret(region, boto_config, assumed_credentials, secret):
    access_key_id = assumed_credentials['AccessKeyId']
    secret_access_key =  assumed_credentials['SecretAccessKey']
    session_token = assumed_credentials['SessionToken']
    session = boto3.Session (aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    scm_client = session.client('secretsmanager', config=boto_config)
    print '\tUsing Caller_id: {}'.format(get_current_id(session, boto_config))
    response = scm_client.get_secret_value(SecretId=secret)
    response.pop('ResponseMetadata')
    return response

def cog_get_user(region, boto_config, assumed_credentials, pool_id, user_name):
    access_key_id = assumed_credentials['AccessKeyId']
    secret_access_key =  assumed_credentials['SecretAccessKey']
    session_token = assumed_credentials['SessionToken']
    session = boto3.Session (aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    print '\tUsing Caller_id: {}'.format(get_current_id(session, boto_config))
    cog_client = session.client('cognito-idp', config=boto_config)
    response = cog_client.admin_get_user(UserPoolId=pool_id, Username=user_name)
    response.pop('ResponseMetadata')
    return response

def list_s3_objects(region, boto_config, session, bucket):
    s3_client = session.client('s3', config=boto_config)
    print '\tUsing Caller_id: {}'.format(get_current_id(session, boto_config))
    response = s3_client.list_objects(Bucket=bucket)
    s3_objects = response.pop('Contents')
    response = []
    for s3_object in s3_objects:
        response.append(s3_object['Key'])
    return response

def ddb_get_item(region, boto_config, dynamo_keys, assume_creds=''):
    access_key_id = assume_creds['AccessKeyId']
    secret_access_key =  assume_creds['SecretAccessKey']
    session_token = assume_creds['SessionToken']
    session = boto3.Session (aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    ddb_client = session.client('dynamodb', config=boto_config)
    print '\tUsing Assumed Caller_id: {}'.format(get_current_id(session, boto_config))
    table = dynamo_keys['table']
    name = dynamo_keys['name']
    email = dynamo_keys['email']
    response = ddb_client.get_item(TableName=table, Key={"name": {"S": name}, "email": {"S": email}})
    response.pop('ResponseMetadata')
    return response

def main(region, profile):
    session = boto3.Session(region_name=region, profile_name=profile)
    boto_config = botocore.config.Config(retries={'max_attempts': 0})
    main_account = '381978683274'
    sub_account = '001566749583'
    local_role_arn = 'arn:aws:iam::001566749583:role/hopper-api-sub-billing-lambda' 
    remote_role_arn = 'arn:aws:iam::381978683274:role/DRGNXATRUST'
    dynamo_keys = {'table':'hopper_dragonchains_dev', 'name':'billing-dev', 'email':'developer@dragonchain.com'}

    print 'Testing X-Account access to: ', main_account

    try:
        print '\nTesting sts.assume_role role: {} in account: {}...'.format(remote_role_arn, main_account)
        assumed_credentials = get_credentials(region, session, boto_config, remote_role_arn)
        print '\tCredentials: {}'.format(assumed_credentials)

        print '\nTesting secretsmanager.get_secret_value in account: {}...'.format(main_account)
        for secret in ['dev/elastic/password', 'githubServiceAccount', 'dev/billing/apikey', 'dev/chain/mnemonic']:
            print '\tSecret: {}'.format(get_secret(region, boto_config, assumed_credentials, secret))

        user_name = 'Developer'
        pool_id = 'us-west-2_EU0PnxwyG'
        print '\nTesting cognito-idp.admin_get_user in account: 381978683274 for user: {} in Pool:{}...'.format(user_name, pool_id)
        user_info = cog_get_user(region, boto_config, assumed_credentials, pool_id, user_name)
        print '\t{}'.format(user_info)

        print '\nTesting s3.list_objects in account: {}...'.format(main_account)
        buckets = ['hopper-global-dev', 'dc6c581fb-e171-4f98-a7fe-serverlessdeploymentbuck-13q9qwp9yz5fa', 'hopper-library-smart-contracts-dev']
        for bucket in buckets:
            objects = list_s3_objects(region, boto_config, session, bucket)
            sys.stdout.write('\tListing S3 Bucket: {}...'.format(bucket))
            sys.stdout.flush()
            print ' Found: {} Objects'.format(size(objects))
        
        print '\nTesting dynamo.get_item in account: {}...'.format(main_account)
        print '\tTesting sts.assume_role role: {} in account: {}...'.format(local_role_arn, sub_account)
        local_creds = get_credentials(region, session, boto_config, local_role_arn)
        print '\tTesting sts.assume_role role: {} in account: {}...'.format(remote_role_arn, main_account)
        remote_creds = get_credentials(region, session, boto_config, remote_role_arn, local_creds)
        print ddb_get_item(region, boto_config, dynamo_keys, remote_creds)

    except Exception as e:
        print '\tERROR: {}'.format(e)

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='initializes baseline dragonchain artifacts')
    parser.add_argument('-r', '--region', nargs='?', default='us-west-2', dest='region', help=' <ec2_region>')
    parser.add_argument('-p', '--profile', nargs='?', default='default', dest='profile', help=' <aws_profile>')
    args = parser.parse_args()
    region = args.region
    profile = args.profile
    main(region, profile)