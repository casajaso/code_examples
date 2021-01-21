#!/usr/bin/env python

#  Author: Jason Casas - Nordstrom Public Cloud
#  python example using custom credential_process to retrieve credentials
#  Requires aws-sdk v1.8.0 or above to support custom credential_process
 
import os
import boto3

def main():
    session = boto3.Session(region_name="us-west-2", profile_name="nordstrom-federated")
    client = session.client('s3')
    for bucket in client.list_buckets()['Buckets']:
        print(bucket)
 
if __name__ == '__main__':
    main()
