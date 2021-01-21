#!/usr/bin/env python3

#  Author: Jason Casas - Nordstrom Public Cloud
#  python example using custom credential_process to retrieve credentials
#  Requires aws-sdk v1.8.0 or above to support custom credential_process


import boto3
import json

if __name__ == '__main__':

    try:
        session = boto3.Session(region_name="us-west-2", profile_name="sso-user")
        client = session.client('sts')
        response = client.get_caller_identity()
        response.pop('ResponseMetadata')
    except Exception as error:
        raise(error)
    print(json.dumps(response, indent=4, sort_keys=True, default=str))
