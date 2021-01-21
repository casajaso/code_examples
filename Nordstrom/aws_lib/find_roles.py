#!/usr/bin/env python3

"""
Author: Jason Casas - Public Cloud
Copyright: Nordstrom 2021
Title: IDP Auditer
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
        json_data = json.dumps(data, indent=4, sort_keys=False, default=str)
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

class AccountAuditer():
    """
    """
    def __init__(self):
        self.errors = []
        self.results = []
        self.log = logger('debug')


    def threader(self, accounts):
        """
        """
        try:
            start_time = datetime.now()
            self.log.info("Evaluating {} accounts with max_workers={}".format(len(accounts), 15))
            with concurrent.futures.ThreadPoolExecutor(max_workers=15) as executor:
                #executor.map(self.get_idps, accounts)
                executor.map(self.get_role_trusts, accounts)
            duration = (datetime.now() - start_time).seconds
            self.log.info("Evaluated {} accounts in {} seconds with max_workers={}".format(len(accounts), duration, 15))
        except Exception as err:
            raise(err)


    def get_idps(self, account_metadata):
        """
        """
        try:     
            idp_metadata = {
                "AccountId": account_metadata.get('accountNumber'),
                "Environment": account_metadata.get('environment'),
                "AccountName": account_metadata.get('accountName'),
                "ADFS": False,
                "OKTA": False,
                "OKTAPCI": False,
                "MISC": []
            }
            account = get_aws_session_cascadelib(account_id=account_metadata.get('accountNumber'), linked=True)
            session = account.session
            iam = session.client('iam')
            response = iam.list_saml_providers()['SAMLProviderList']
            if response:
                for idp in response:
                    if idp.get('Arn'):
                        if idp.get('Arn') == 'arn:aws:iam::{}:saml-provider/NORD'.format(account_metadata.get('accountNumber')):
                            idp_metadata['ADFS'] = True
                        elif idp.get('Arn') == 'arn:aws:iam::{}:saml-provider/OKTA'.format(account_metadata.get('accountNumber')):
                            idp_metadata['OKTA'] = True
                        elif idp.get('Arn') == 'arn:aws:iam::{}:saml-provider/OKTAPCI'.format(account_metadata.get('accountNumber')):
                            idp_metadata['OKTAPCI'] = True
                        else:
                            idp_metadata['MISC'].append(idp['Arn'].split('/')[1])
                    print('found ({}) Identity providors: '.format(len(response)))
            self.results.append(idp_metadata)
            if not idp_metadata.get('OKTA'):
                self.log.warn('AccountID: {} is missing primary OKTA SAML Provider! IdpMetadata: ({})'.format(account_metadata.get('accountNumber'), idp_metadata))
            if idp_metadata.get('Environment') in ['PCIProd', 'PCINonProd'] and idp_metadata.get('ADFS'):
                self.log.warn('ADFS providor is configured for PCI AccountID: {}! IdpMetadata: ({})'.format(account_metadata.get('accountNumber'), idp_metadata))
            if idp_metadata.get('Environment') not in ['PCIProd', 'PCINonProd'] and idp_metadata.get('OKTAPCI'):
                self.log.warn('OKTA PCI providor is configured for Non-PCI AccountID: {}! IdpMetadata: ({})'.format(account_metadata.get('accountNumber'), idp_metadata))
        except Exception as err:
            raise(err)   

    def get_role_trusts(self, account_metadata):
        """
        """
        try:     
            account = get_aws_session_cascadelib(account_id=account_metadata.get('accountNumber'), linked=True)
            session = account.session
            iam = session.client('iam')
            response = paginate_results(iam, 'list_roles')
            roles = list(filter(lambda _role: (
                # 'SRE-Team' in _role['RoleName']
                # or 'DevUsers-Team' in _role['RoleName']
                '_DevUsers' in _role['RoleName'] and '_DevUsers-Team' not in _role['RoleName']
                or '_CloudEng' in _role['RoleName'] and '_CloudEng-Team' not in _role['RoleName']
                or '_CloudSec' in _role['RoleName'] and '_CloudSec-Team' not in _role['RoleName'] and '_CloudSecurity' not in _role['RoleName']
            ), response['Roles']))
            for _role in roles:
                role_trust_metadata = {
                    'AccountId': account_metadata['accountNumber'],
                    "RoleName": _role['RoleName'],
                    "Environment": account_metadata.get('environment'),
                    "AccountName": account_metadata.get('accountName'),
                    'ADFS': False,
                    'OktaSAML': False,
                    'OktaSAMLPCI': False,
                    'OktaOIDC': False,
                    'OktaOIDCFFF000': False,
                    'MissConfigured': [],
                    'Extra': []
                }
                for policy_statement in _role['AssumeRolePolicyDocument']['Statement']:
                    principals = policy_statement['Principal']
                    _keys = principals.keys()
                    print('keys: {}'.format(_keys))
                    for _key in _keys:
                        identities = principals.get(_key)
                        if isinstance(identities, str):
                            identities = [identities]
                        for principal in identities:
                            if principal == 'arn:aws:iam::{}:saml-provider/NORD'.format(account_metadata.get('accountNumber')):
                                role_trust_metadata['ADFS'] = True
                            elif principal == 'arn:aws:iam::{}:saml-provider/OKTA'.format(account_metadata.get('accountNumber')):
                                role_trust_metadata['OktaSAML'] = True
                            elif principal == 'arn:aws:iam::{}:saml-provider/OKTAPCI'.format(account_metadata.get('accountNumber')):
                                role_trust_metadata['OktaSAMLPCI'] = True
                            elif principal == 'arn:aws:iam::028824555316:role/aws-okta-tokensvc':
                                role_trust_metadata['OktaOIDC'] = True
                            elif principal == 'arn:aws:iam::028824555316:role/fff000-Okta-Token-Service':
                                role_trust_metadata['OktaOIDCFFF000'] = True
                            elif 'saml-provider' in principal and account_metadata.get('accountNumber') not in principal:
                                role_trust_metadata['MissConfigured'].append(principal)
                            else:
                                if '028824555316:root' in principal:
                                    role_trust_metadata['Extra'].append('IEPC-PROD')
                                elif '125237321655:root' in principal:
                                    role_trust_metadata['Extra'].append('IEPC-NONPROD')
                                elif '908505540520:root' in principal:
                                    role_trust_metadata['Extra'].append('CSEC-NONPROD')
                                elif '584740350566:root' in principal:
                                    role_trust_metadata['Extra'].append('CSEC-PCIPROD')
                                elif '563390611068:root' in principal:
                                    role_trust_metadata['Extra'].append('CSEC-PCINONPROD')
                                elif '165387580306:root' in principal:
                                    role_trust_metadata['Extra'].append('CSEC-PROD')
                                else:
                                    role_trust_metadata['Extra'].append(principal)    
                print('role_trust_metadata: {}'.format(role_trust_metadata))
                self.results.append(role_trust_metadata)
                # if not role_trust_metadata.get('OktaSAML') and 'PCI' not in role_trust_metadata.get('Environment'):
                #     self.log.warn('AccountID: {} Role: {} is missing trust with primary OKTA SAML Provider! IdpMetadata: ({})'.format(role_trust_metadata.get('RoleName'), role_trust_metadata.get('AccountId'), role_trust_metadata))
                # if not role_trust_metadata.get('OktaSAMLPCI') and 'PCI' in role_trust_metadata.get('Environment'):
                #     self.log.warn('AccountID: {} Role: {} is missing trust with primary PCI OKTA SAML Provider! IdpMetadata: ({})'.format(role_trust_metadata.get('RoleName'), role_trust_metadata.get('AccountId'), role_trust_metadata))
                if not role_trust_metadata.get('OktaOIDC'):
                    self.log.warn('AccountID: {} Role: {} is missing trust with primary OKTA OIDC Provider! IdpMetadata: ({})'.format(role_trust_metadata.get('AccountId'), role_trust_metadata.get('RoleName'), role_trust_metadata))
        except Exception as err:
            raise(err) 
    

def get_accounts(scope='all', metadata=False): 
    '''
    '''
    accounts = []
    all = ['nonprod', 'pcinonprod', 'prod', 'pciprod']
    assert (scope.lower() in ['all'] + all), AssertionError(
        "Scope must be one of: {} recieved: ({})".format(['all'] + all, scope)
    )
    if scope.lower() == 'all':
        target_env = all
    else: 
        target_env = [scope]
    try:
        accounts_meta = FirestoreHelper().query_r('aws-accounts')['data']
        for env in target_env:
            for account in accounts_meta:
                if account['active'] and account['environment'].lower() == env.lower():
                    if metadata:
                        accounts.append(account)
                    else:
                        accounts.append(account.get('accountNumber'))
    except Exception as err:
        raise (err) 
    assert (len(list(accounts)) > 0), AssertionError(" Unable to retrieve accounts from firestore")
    return accounts

def get_account_metadata(account_number):
    '''
    '''
    try:
        response = FirestoreHelper().query_r('aws-accounts', account_number)['data']
    except Exception as err:
        raise (err)
    return response

def main():
    log = logger('debug')
    log.debug('Begin..')
    idps = []
    try:
        base_session = get_aws_session_cascadelib(linked=False)
        #accounts = [get_account_metadata("xxxxxxxxxxxxxxxx"), get_account_metadata("xxxxxxxxxxxxxxxx")] #Sandbox Testing
        accounts = get_accounts(scope='All', metadata=True)
        audit = AccountAuditer()
        audit.threader(accounts)
        write_dict_to_csv_file(audit.results, 'roles')
        log.debug('recorded ({}) records'.format(len(audit.results)))
        print(jsonify(audit.results))
    except Exception as err:
        log.error(err, exc_info=True)
        os._exit(1)

if __name__ == '__main__':
    main()