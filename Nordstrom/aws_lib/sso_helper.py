#!/usr/bin/env python3

import json
import csv
import os
import boto3
import logging
import sys
from datetime import datetime
from datetime import timedelta
from datetime import timezone
from pathlib import Path
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
        logger = logging.getLogger(sys.argv[0])
        logger.setLevel(log_level.upper())
        stream_handler = logging.StreamHandler(sys.stdout)
        file_handler = logging.FileHandler(log_file)
        stream_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
        file_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
        logger.addHandler(stream_handler)
        logger.addHandler(file_handler)
        logger.propagate = False
    except Exception as err:
        raise (err)
    return logger

def assign_account(session, sso_instance_arn, account_id, permission_set, principle_type, principle_id):
    """
    """
    try:
        ssoadmin = session.client('sso-admin', region_name='us-west-2')
        response = ssoadmin.create_account_assignment(
            InstanceArn=sso_instance_arn,
            TargetId=account_id,
            TargetType='AWS_ACCOUNT',
            PermissionSetArn=permission_set,
            PrincipalType=principle_type.upper(),
            PrincipalId=principle_id
        )
        response.pop('ResponseMetadata')
    except Exception as err:
        raise(err)
    return response

def get_sso_permission_set_info(session, sso_instance_arn, permission_set=None):
    """
    """
    try:
        ssoadmin = session.client('sso-admin', region_name='us-west-2')
        if permission_set:
            response = ssoadmin.describe_permission_set(
                InstanceArn=sso_instance_arn, 
                PermissionSetArn=permission_set
            )
            response.pop('ResponseMetadata')
        else:
            response = []
            sets = ssoadmin.list_permission_sets(InstanceArn=sso_instance_arn)
            sets.pop('ResponseMetadata')
            for _permission_set in sets.get('PermissionSets'):
                permission_set_md = ssoadmin.describe_permission_set(
                    InstanceArn=sso_instance_arn, 
                    PermissionSetArn=_permission_set
                )
                permission_set_md.pop('ResponseMetadata')
                response.append(permission_set_md)
    except Exception as err:
        raise(err)
    return response

def get_account_assignments(session, sso_instance_arn, account_id, permission_set):
    """
    """
    try:
        ssoadmin = session.client('sso-admin', region_name='us-west-2')
        response = ssoadmin.list_account_assignments(
            InstanceArn=sso_instance_arn,
            AccountId=account_id,
            PermissionSetArn=permission_set,
        )
        response.pop('ResponseMetadata')
    except Exception as err:
        raise(err)
    return response

def check_assignments(session, sso_instance_arn, account_id, permission_set, principle_id):
    """
    """
    assignments = get_account_assignments(
        session, 
        sso_instance_arn, 
        account_id, 
        permission_set
    ).get('AccountAssignments')
    response = False
    for _assignment in assignments:
        if _assignment['PrincipalId'] == principle_id and _assignment['PermissionSetArn'] == permission_set:
            response = True
    return response

def check_organization(session, account_id):
    """
    """
    orgs = boto3.client('organizations')
    response = orgs.describe_account(
        AccountId=account_id
    )
    if not response.get('Account'):
        return False
    else: 
        return response.get('Account')

def main():
    log = logger()
    sso_instance_arn_prod = 'arn:aws:sso:::instance/ssoins-xxxxxxxxxxxxxxxx'
    sso_instance_arn_moon = 'arn:aws:sso:::instance/ssoins-xxxxxxxxxxxxxxxx'
    sso_permission_set_arn_prod_administratoraccess = 'arn:aws:sso:::permissionSet/ssoins-xxxxxxxxxxxxxxxx/ps-xxxxxxxxxxxxxxxx'
    sso_permission_set_arn_moon_administratoraccess = 'arn:aws:sso:::permissionSet/ssoins-xxxxxxxxxxxxxxxx/ps-xxxxxxxxxxxxxxxx'
    sso_permission_set_arn_prod_viewonlyaccess = 'arn:aws:sso:::permissionSet/ssoins-xxxxxxxxxxxxxxxx/ps-xxxxxxxxxxxxxxxx'
    sso_permission_set_arn_moon_viewonlyaccess = 'arn:aws:sso:::permissionSet/ssoins-xxxxxxxxxxxxxxxx/ps-xxxxxxxxxxxxxxxx'
    jason='xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    nordcloudadmin='xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    TECH_CXPAWS='xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    TechnologyPublicCloudViewOnlyProd="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    TechnologyPublicCloudViewOnlyMoon="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    NordCloudAdminsMoon="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    artemis='xxxxxxxxxxxxxxxx'

    # prod
    # instance_arn = sso_instance_arn_prod
    # perm_set = sso_permission_set_arn_prod_viewonlyaccess
    # group_name = TechnologyPublicCloudViewOnlyProd
    # input_file = open('assign-accounts.txt', 'r')
    # firestore_helper = FirestoreHelper()
    # session = boto3.Session(profile_name='billing')

    # moon
    instance_arn = sso_instance_arn_moon
    perm_set = sso_permission_set_arn_moon_viewonlyaccess
    group_name = TechnologyPublicCloudViewOnlyMoon
    input_file = open('assign-accounts-moon.txt', 'r')
    session = boto3.Session(profile_name='moon')

    # moon admin
    # instance_arn = sso_instance_arn_moon
    # perm_set = sso_permission_set_arn_moon_administratoraccess
    # group_name = NordCloudAdminsMoon
    # input_file = open('assign-accounts-moon.txt', 'r')
    # session = boto3.Session(profile_name='moon')

    account_ids = input_file.readlines() 

    for _account_id in account_ids:
        account_id = _account_id.replace('\n', '')
        # if not account_md:
        #     log.warning('unable to locate account metadata for AccountId: {}'.format(account_id))
        #     in_org = check_organization(session, account_id)
        #     log.warning('orgs_md: {}'.format(in_org)) # retruns :/ botocore.errorfactory.AccessDeniedException: An error occurred (AccessDeniedException) when calling the DescribeAccount operation: You don't have permissions to access this resource.
        if instance_arn == 'arn:aws:sso:::instance/ssoins-7907885ec03a3c55':
            log.info("proccessing accountNumber: {}".format(account_id))
            is_assigned = check_assignments(session, instance_arn, account_id, perm_set, group_name)
            if not is_assigned:
                log.info("Assigning with peramaeters session: {} ssoInstanceArn: {} accountId: {} permissionSetArn: {} principleType: {} principleId: {}".format(session, instance_arn, account_id, perm_set, 'group', group_name))
                response = assign_account(session, instance_arn, account_id, perm_set, 'group', group_name)
                log.info(response)
            else:
                log.error('Assignment already exists')
        else:
            account_md = firestore_helper.query_r('aws-accounts', account_id).get('data')
            if 'pci' in account_md.get('environment').lower():
                log.error("cannot assign sso access for AccountId: {} because it is {}".format(account_id, account_md.get('environment').lower()))
                sys.exit(1)
            else:
                log.info("proccessing accountNumber: {}, accountName: {}, environment {}".format(account_md['accountNumber'], account_md['accountName'], account_md['environment']))
                is_assigned = check_assignments(session, instance_arn, account_md['accountNumber'], perm_set, group_name)
                if not is_assigned:
                    log.info("Assigning with peramaeters session: {} ssoInstanceArn: {} accountId: {} permissionSetArn: {} principleType: {} principleId: {}".format(session, instance_arn, account_md['accountNumber'], perm_set, 'group', group_name))
                    response = assign_account(session, instance_arn, account_md['accountNumber'], perm_set, 'group', group_name)
                    log.info(response)
                else:
                    log.error('Assignment already exists')

if __name__ == '__main__':
    main()