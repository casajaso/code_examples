#!/usr/bin/env python3

import boto3
import os


account_metadata = {'accountNumber': 'xxxxxxxxxxxxxxxx'}
try:
    session = boto3.Session(profile_name='billing')
    iam = session.client('iam')
    response = iam.list_roles()

    roles = list(filter(lambda _role: (
        'SRE-Team' in _role['RoleName']
        or 'DevUsers-Team ' in _role['RoleName']
        or 'DevUsers' in _role['RoleName'] 
        or 'CloudEng' in _role['RoleName'] 
        or 'CloudSec' in _role['RoleName']
    ), response['Roles']))
    _r = []
    for _role in roles:
        role_trust_metadata = {
            'AccountId': account_metadata['accountNumber'],
            "RoleName": _role['RoleName'],
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
            for _key in _keys:
                identities = principals.get(_key)
                if isinstance(identities, str):
                    identities = [identities]
                for principal in identities:
                    if principal == 'arn:aws:iam::{}:saml-provider/xxxxxxxxxxxxxxxx'.format(account_metadata.get('accountNumber')):
                        role_trust_metadata['ADFS'] = True
                    elif principal == 'arn:aws:iam::{}:saml-provider/xxxxxxxxxxxxxxxx'.format(account_metadata.get('accountNumber')):
                        role_trust_metadata['OktaSAML'] = True
                    elif principal == 'arn:aws:iam::{}:saml-provider/xxxxxxxxxxxxxxxx'.format(account_metadata.get('accountNumber')):
                        role_trust_metadata['OktaSAMLPCI'] = True
                    elif principal == 'arn:aws:iam::xxxxxxxxxxxxxxxx:role/xxxxxxxxxxxxxxxx':
                        role_trust_metadata['OktaOIDC'] = True
                    elif principal == 'arn:aws:iam::xxxxxxxxxxxxxxxx:role/xxxxxxxxxxxxxxxx':
                        role_trust_metadata['OktaOIDCFFF000'] = True
                    elif 'saml-provider' in principal and account_metadata.get('accountNumber') not in principal:
                        role_trust_metadata['MissConfigured'].append(principal)
                    else:
                        role_trust_metadata['Extra'].append(principal)
        _r.append(role_trust_metadata)
except Exception as err:
    raise(err) 
    os._exit(1) 
print('results: {}'.format(_r))
