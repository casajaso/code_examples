#!/usr/bin/env python3

#  Author: Jason Casas - Nordstrom Public Cloud
#  python example using sso to retrieve credentials
#  Requires aws-sdk v or above to support sso.get_role_credentials
 
import os
import boto3
import json

def get_access_token():
    dir = os.path.expanduser('~/.aws/sso/cache')
    files = os.listdir(dir)
    for file in files:
        if file.endswith('.json') and (len(file.split('.')[0])) == int(40):
            with open('{}/{}'.format(dir, file), "r") as input:
                data = json.loads(input.read())
            if 'accessToken' and 'region' in data:
                return(data['accessToken'], data['region'])
    raise ValueError('Unable to retrieve accessToken in sso_cache')

def get_credentials(profile, account_id, role):
    try:
        access_token, sso_region = get_access_token()
        session = boto3.Session(profile_name=profile)
        sso = session.client('sso', region_name=sso_region)
        response = sso.get_role_credentials(
            roleName=role,
            accountId=account_id,
            accessToken=access_token
        )
        credentials = response['roleCredentials']
    except Exception as error:
        raise error
    return credentials
    
def main():
    try:
        credentials = get_credentials("sso-user", '116019673048', 'AdministratorAccess')
        session = boto3.Session(
            region_name="us-west-1", 
            aws_access_key_id=credentials['accessKeyId'], 
            aws_secret_access_key=credentials['secretAccessKey'], 
            aws_session_token=credentials['sessionToken']
        )
        sts = session.client('sts')
        response = sts.get_caller_identity()
        response.pop('ResponseMetadata')
    except Exception as error:
        raise(error)
        
    print(json.dumps(response, indent=4, sort_keys=True, default=str))

if __name__ == '__main__':
    main()
