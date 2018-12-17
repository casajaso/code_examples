#!/usr/bin/env python
# 08/2018
# Provisions full baseline infrustructure stack in target AWS account (multi/sub account and/or cross region)

import os
import sys
import boto3
import botocore.config
import argparse
import io
import yaml
import json

# [utility functions]
def convert_template(cf_template):
    with io.open(cf_template, 'r', encoding='utf8') as yaml_cft:
        data = yaml.load(yaml_cft)
        json_cft_body = (json.dumps(data, sort_keys=True, indent=4, ensure_ascii=False, encoding='utf8'))
    return json_cft_body

def get_current_id(session, boto_config): #CALL THIS FUNCT TO FIND OUT WHO YOU ARE WHEN DEBUGGING
    sts_client = session.client('sts', config=boto_config)
    response = sts_client.get_caller_identity()
    response.pop('ResponseMetadata')
    return response

def get_credentials(session, boto_config, role_arn, credentials=''):
    if credentials != '':
        access_key_id = credentials['AccessKeyId']
        secret_access_key =  credentials['SecretAccessKey']
        session_token = credentials['SessionToken']
        session = boto3.Session(aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    sts_client = session.client('sts', config=boto_config)
    try:
        assumed_role_object = sts_client.assume_role(RoleArn=role_arn, RoleSessionName='Bootstrap_assumed_role')
        response = assumed_role_object['Credentials']
        return response
    except Exception as error:
        print '[ERROR][GET CREDENTIALS]: {}'.format(error)

# [create base resources]
def create_cmk(session, boto_config, encryption_key):
    kms_client = session.client('kms', config=boto_config)
    encryption_key_arn = 'alias/drgn/IAM/KeyPair/{}'.format(encryption_key)
    try:
        cmk_arn = (kms_client.describe_key(KeyId=encryption_key_arn)['KeyMetadata']['Arn'])
        print 'Found EncrKey: {}'.format(cmk_arn)
    except kms_client.exceptions.NotFoundException as error:
        sys.stdout.write('Creating encryption key...')
        sys.stdout.flush()
        response = kms_client.create_key(Description=encryption_key, KeyUsage='ENCRYPT_DECRYPT')
        cmk_arn = response['KeyMetadata']['Arn']
        kms_client.create_alias(AliasName=encryption_key_arn, TargetKeyId=cmk_arn)
        print ' Created KeyId: {}'.format(response['KeyMetadata']['Arn'])

def create_keypair(session, boto_config, key_name, encryption_key):
    ec2_client = session.client('ec2', config=boto_config)
    try:
        keypair = ec2_client.describe_key_pairs(KeyNames=[key_name])['KeyPairs'][0]
        print 'Found KeyPair: {}'.format(keypair)
    except ec2_client.exceptions.ClientError as error:
        sys.stdout.write('Creating Keypair: {}...'.format(key_name))
        sys.stdout.flush()
        response = ec2_client.create_key_pair(KeyName=key_name)
        secret_string = response['KeyMaterial']
        response.pop('ResponseMetadata')
        response.pop('KeyMaterial')
        print ' Created KeyPair: {}'.format(response)
        create_secret(session, boto_config, secret_string, key_name, encryption_key)

def create_secret(session, boto_config, secret_string, secret_name, encryption_key):
    scm_client = session.client('secretsmanager', config=boto_config)
    encryption_key_arn = 'alias/drgn/IAM/KeyPair/{}'.format(encryption_key)
    description = '{}'.format(secret_name)
    try:
        secret_arn = (scm_client.describe_secret(SecretId=secret_name)['ARN'])
        response = scm_client.update_secret(SecretId=secret_name, Description=description, KmsKeyId=encryption_key_arn, SecretString=secret_string)
        sys.stdout.write('Updating Secret: {}...'.format(secret_name))
        sys.stdout.flush()
    except scm_client.exceptions.ResourceNotFoundException:
        sys.stdout.write('Creating Secret: {}...'.format(secret_name))
        sys.stdout.flush()
        response = scm_client.create_secret(Name=secret_name, Description=description, KmsKeyId=encryption_key_arn, SecretString=secret_string)
    print ' Stored Secret: {}'.format(response['ARN'])

def deploy_cf_stack(session, boto_config, stack_name):
    cf_client = session.client('cloudformation', config=boto_config)
    try:
        stack_info = cf_client.describe_stacks(StackName=stack_name)['Stacks'][0]
        print 'Found StackId: {}'.format(stack_info['StackId'])
    except cf_client.exceptions.ClientError as error:
        sys.stdout.write('Creating cloudformation stack: {}...'.format(stack_name))
        sys.stdout.flush()
        json_cft_body = convert_template('bootstrap_cft.yml')
        try:
            response = cf_client.create_stack(StackName=stack_name, TemplateBody=json_cft_body, Capabilities=['CAPABILITY_NAMED_IAM'])
            waiter = cf_client.get_waiter('stack_create_complete')
            waiter.wait(StackName=stack_name, WaiterConfig={'Delay': 10, 'MaxAttempts': 120})
            response.pop('ResponseMetadata')
            print ' Created StackId: {}'.format(response['StackId'])
        except Exception as error:
            print '[ERROR][DEPLOY STACK]: {}'.format(error)

# [get/clone resources]
def get_secret(session, region, boto_config, secret_name, credentials=''):
    print 'Retrieving Secret Value: {}...'.format(secret_name)
    if credentials != '':
        access_key_id = credentials['AccessKeyId']
        secret_access_key =  credentials['SecretAccessKey']
        session_token = credentials['SessionToken']
        session = boto3.Session(aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    scm_client = session.client('secretsmanager', config=boto_config)
    print 'Using CallerID: {}'.format(get_current_id(session, boto_config)['Arn'])
    try:
        response = scm_client.get_secret_value(SecretId=secret_name)
        response.pop('ResponseMetadata')
        print 'Got Secret Value for: {}'.format(response['ARN']) 
        return json.dumps(json.loads(response['SecretString']))
    except Exception as error:
        print '[ERROR][GET SECRET]: {}'.format(error)

def get_cognito_user(session, region, boto_config, pool_id, user_name, credentials=''):
    print 'Doing a thing: {}...'.format(pool_id)
    if credentials != '':
        access_key_id = assumed_credentials['AccessKeyId']
        secret_access_key =  assumed_credentials['SecretAccessKey']
        session_token = assumed_credentials['SessionToken']
        session = boto3.Session (aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    cog_client = session.client('cognito-idp', config=boto_config)
    response = cog_client.admin_get_user(UserPoolId=pool_id, Username=user_name)
    response.pop('ResponseMetadata')
    return response

def get_s3_objects(session, region, boto_config, bucket_name, credentials=''):
    print 'Doing a thing: {}...'.format(bucket_name)
    if credentials != '':
        access_key_id = credentials['AccessKeyId']
        secret_access_key =  credentials['SecretAccessKey']
        session_token = credentials['SessionToken']
        session = boto3.Session(aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    s3_client = session.client('s3', config=boto_config)
    response = s3_client.list_objects(Bucket=bucket_name)
    s3_objects = response.pop('Contents')
    response = []
    for s3_object in s3_objects:
        response.append(s3_object['Key'])
    return response

def get_ddb_items(session, region, boto_config, dynamo_keys, credentials=''):
    print 'Doing a thing: {}...'.format(pool_id)
    if credentials != '':
        access_key_id = credentials['AccessKeyId']
        secret_access_key =  credentials['SecretAccessKey']
        session_token = credentials['SessionToken']
        session = boto3.Session (aws_access_key_id=access_key_id, aws_secret_access_key=secret_access_key, aws_session_token=session_token, region_name=region)
    ddb_client = session.client('dynamodb', config=boto_config)
    table = dynamo_keys['table']
    name = dynamo_keys['name']
    email = dynamo_keys['email']
    response = ddb_client.get_item(TableName=table, Key={"name": {"S": name}, "email": {"S": email}})
    response.pop('ResponseMetadata')
    return response

# [main]
def main(region, profile): 
    session = boto3.Session(region_name=region, profile_name=profile)
    boto_config = botocore.config.Config(retries={'max_attempts': 0})
    current_account = get_current_id(session, boto_config)['Account']
    current_account_role_arn = get_current_id(session, boto_config)['Arn']
    source_account = 'xxxxxxxxxxxx' #Currently Set to Main AccountId
    source_account_role_arn = 'arn:aws:iam::xxxxxxxxxxxx:role/drgnTrustSub'

    encryption_key = 'drgnEncryptionKey'
    create_cmk(session, boto_config, encryption_key)
    keypairs = ['drgnDefaultKeyPair', 'drgnBastionKeyPair']
    for key_name in keypairs:
            create_keypair(session, boto_config, key_name, encryption_key)
    credentials = get_credentials(session, boto_config, source_account_role_arn)
    secret_name = 'dev/elastic/password'
    secret_string = get_secret(session, region, boto_config, secret_name, credentials)
    create_secret(session, boto_config, secret_string, secret_name, encryption_key)

    stack_name = 'drgnStack'
    deploy_cf_stack(session, boto_config, stack_name)

#Future Add(s) 
#get/replicate cognito users
#    user_name = 'Developer'
#    pool_id = 'us-west-2_EU0PnxwyG'
#    user_info = get_cognito_user(session, region, boto_config, pool_id, user_name, credentials='')
#    print user_info
#get/replicate s3 objects
#    buckets = ['hopper-global-dev', 'bucket', 'hopper-library-smart-contracts-dev']
#    for bucket_name in buckets:
#        objects = get_s3_objects(session, region, boto_config, bucket_name, credentials='')
#        sys.stdout.write('\tListing S3 Bucket: {}...'.format(bucket_name))
#        sys.stdout.flush()
#        print ' Found: {} Objects'.format(size(objects))
#get/replicate dynamodb tables

# [args]
if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='initializes baseline dragonchain artifacts')
    parser.add_argument('-r', '--region', nargs='?', default='us-west-2', dest='region', help=' <ec2_region>')
    parser.add_argument('-p', '--profile', nargs='?', default='default', dest='profile', help=' <aws_profile>')
    args = parser.parse_args()
    region = args.region
    profile = args.profile
    main(region, profile)