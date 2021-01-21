#!/usr/bin/env python3

"""
Author: Jason Casas - Public Cloud
Copyright: Nordstrom 2020
Title:  
Discription: (SEE README.md for more info)
Requires Input: 
"""

import os
import argparse
import boto3
import botocore.exceptions
import copy
import csv
import json
import logging
import concurrent.futures
import sys
from datetime import datetime
from datetime import timedelta
from pathlib import Path
from cascadelib3.account import Account
from cascadelib3.firestore_helper import FirestoreHelper

def logger(log_level='Info'):
    assert (log_level.upper() in ['DEBUG', 'TRACE', 'INFO', 'WARN', 'WARNING', 'ERROR']), AssertionError(
        "Log Level must be one of: {} recieved: ({})".format(log_level)
    )
    try:
        _filename = datetime.now().strftime('{}-%Y-%m-%d_%H-%M-%S.log'.format(sys.argv[0].split('.')[1]))
        _dir = 'logs'
        if not Path(_dir).exists():
            Path(_dir).mkdir(parents=True)
        log_file = '{}/{}'.format(_dir, _filename)
        _logger = logging.getLogger(sys.argv[0])
        _logger.setLevel(log_level.upper())
        stream_handler = logging.StreamHandler(sys.stdout)
        file_handler = logging.FileHandler(log_file)
        stream_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
        file_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
        _logger.addHandler(stream_handler)
        _logger.addHandler(file_handler)
        _logger.propagate = False
    except Exception as err:
        raise (err)
    return _logger

def jsonify(data):
    """
    """
    try:
        json_data = json.dumps(data, indent=4, sort_keys=True, default=str)
    except Exception as err:
        raise(err)
    return json_data

def write_dict_to_csv_file(input, filename):
    """
    """
    try:
        _filename = '{}.csv'.format(filename)
        _dir = 'results'
        if not Path(_dir).exists():
            Path(_dir).mkdir(parents=True)
        csv_file = open('{}/{}'.format(_dir, _filename), 'w')
        row = 0
        csv_writer = csv.writer(csv_file)
        for _entry in input:
            if row == 0:
                header = _entry.keys()
                csv_writer.writerow(header) 
                row += 1
            csv_writer.writerow(_entry.values()) 
        csv_file.close()
    except Exception as err:
        raise(err) 

def get_aws_session_cascadelib(account_id=None, linked=True, target_role_arn=None, profile=None, region='us-west-2'):
    """
    """
    try:
        account = Account(
            account=account_id, 
            linked=linked, 
            target_role_arn=target_role_arn, 
            profile=profile, 
            region=region
        )
        assert (account.is_valid()), AssertionError("Unable to assume role")
        session = account.session
        sts = session.client('sts')
        id = sts.get_caller_identity()
        id.pop('ResponseMetadata')
        print('authenticated: ({})'.format(id['Arn']))
    except AttributeError as err:
        if str(err) == "'NoneType' object has no attribute 'client'":
            raise ChildProcessError("downstream exception encountered: cascadelib3.get_linked_session: {}: ({})".format(
                err, "likely assume role permission issue")
            ) from err
    except Exception as err:
        raise (err) 
    return account

def get_vpc_regions(session):
    """
    """
    try:
        search_regions = ['us-east-1', 'us-east-2', 'us-west-1', 'us-west-2']
        active_regions = []
        for region in search_regions:
            ec2 = session.client('ec2', region_name=region)
            response = ec2.describe_vpcs()['Vpcs']
            if len(response) > 0:
                active_regions.append(region)
    except Exception as err:
        raise(err)      
    return active_regions

def paginate_results(client, operation, config={}):
    """
    """
    try:
        assert client.can_paginate(operation), Exception(" operation does not support pagination: {}".format(operation))
        paginator = client.get_paginator(operation)
        response = paginator.paginate(PaginationConfig=config).build_full_result()
    except Exception as err:
        raise(err)          
    return response

def audit_instances(session, search_regions, account_id):
    """
    """
    try:
        response = []
        for region in search_regions:
            _r = []
            ec2 = session.client('ec2', region_name=region)
            print('searching region: {}'.format(region))
            reservations = paginate_results(ec2, 'describe_instances')['Reservations'] 
            for reservation in reservations:
                if reservation.get("Instances"):
                    for instance in reservation["Instances"]:
                        _image = ec2.describe_images(ImageIds=[instance['ImageId']])
                        if len(_image['Images']) > 0:
                            image = _image['Images'][0]
                        else:
                            image = {}
                        instance_metadata = {
                            "AccountId" : account_id,
                            "Region" : region,
                            "InstanceId": instance['InstanceId'],
                            "ImageId": instance['ImageId'],
                            "Platform": instance.get('Platform'),
                            "PlatformDetails": image.get('PlatformDetails'),
                            "PlatformDescription": image.get('Description')
                            # "InstanceTags": instance.get('Tags'),
                            # "ImageTags": image.get('Tags')
                        }
                        _r.append(instance_metadata)
            print('found ({}) instances: '.format(len(_r)))
            response = response + _r
    except Exception as err:
        raise(err)   
    return response

def audit_security_groups(session, search_regions, account_id):
    """
    """
    try:
        response = []  
        for region in search_regions:
            _r = []
            ec2 = session.client('ec2', region_name=region)
            print('searching region: {}'.format(region))
            sec_groups = paginate_results(ec2, 'describe_security_groups')['SecurityGroups']
            for sec_group in sec_groups:
                ingress_rules = []
                egress_rules = []
                for rule in sec_group['IpPermissions']:
                    _rule = {"Port": "", "Source":[]}
                    if rule['IpProtocol'] == '-1':
                        _rule['Port'] = 'all'
                    elif rule.get("FromPort", None) !=  None:
                        _rule['Port'] = '{} - {}'.format(rule["FromPort"], rule["ToPort"])
                    if len(rule['IpRanges']) != 0:
                        for item in rule["IpRanges"]:
                            _rule["Source"].append(item["CidrIp"])
                    if len(rule['Ipv6Ranges']) != 0:
                        for item in rule["Ipv6Ranges"]:
                            _rule["Source"].append(item["CidrIpv6"])
                    if len(rule['UserIdGroupPairs']) != 0:
                        for item in rule["UserIdGroupPairs"]:
                            _rule["Source"].append(item["GroupId"])
                    ingress_rules.append(_rule)
                for rule in sec_group['IpPermissionsEgress']:
                    _rule = {"Port": "", "Source":[]}
                    if rule['IpProtocol'] == '-1':
                        _rule['Port'] = 'all'
                    elif rule.get("FromPort", None) !=  None:
                        _rule['Port'] = '{} - {}'.format(rule["FromPort"], rule["ToPort"])
                    if len(rule['IpRanges']) != 0:
                        for item in rule["IpRanges"]:
                            _rule["Source"].append(item["CidrIp"])
                    if len(rule['Ipv6Ranges']) != 0:
                        for item in rule["Ipv6Ranges"]:
                            _rule["Source"].append(item["CidrIpv6"])
                    if len(rule['UserIdGroupPairs']) != 0:
                        for item in rule["UserIdGroupPairs"]:
                            _rule["Source"].append(item["GroupId"])
                    egress_rules.append(_rule)
                sec_group_metadata = {
                    "AccountId" : account_id,
                    "Region" : region,
                    "VpcId" : sec_group['VpcId'],
                    "GroupId" : sec_group['GroupId'],
                    "GroupName" : sec_group['GroupName'],
                    "IngressRules" : ingress_rules,
                    "EgressRules" : egress_rules,
                    "Description" : sec_group['Description']
                }
                _r.append(sec_group_metadata)
            print('found ({}) security groups: '.format(len(_r)))
            response = response + _r
    except Exception as err:
        raise(err)   
    return response


def main():
    log = logger('debug')
    log.debug('Begin..')
    ec2_insgances = []
    ec2_secuirty_groups = []
    try:
        base_session = get_aws_session_cascadelib(linked=False)
        accounts = ["xxxxxxxxxxxxxxxx", "xxxxxxxxxxxxxxxx", "xxxxxxxxxxxxxxxx", "xxxxxxxxxxxxxxxx"]
        for account_id in accounts:
            account = get_aws_session_cascadelib(account_id=account_id, linked=True)
            session = account.session
            log.debug('searching account_id: {}'.format(account_id))
            regions = get_vpc_regions(session)
            log.debug('searching regions: {}'.format(regions))
            _security_groups = audit_security_groups(session, regions, account_id)
            ec2_secuirty_groups = ec2_secuirty_groups + _security_groups
            _instances = audit_instances(session, regions, account_id)
            ec2_insgances = ec2_insgances + _instances
        write_dict_to_csv_file(ec2_secuirty_groups, 'security_groups')
        log.debug('recorded ({}) security groups'.format(len(ec2_secuirty_groups)))
        write_dict_to_csv_file(ec2_insgances, 'instances')
        log.debug('recorded ({}) instances'.format(len(ec2_insgances)))
    except Exception as err:
        log.error(err, exc_info=True)
        os._exit(1)

if __name__ == '__main__':
    main()