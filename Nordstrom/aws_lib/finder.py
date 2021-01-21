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
import json
import logging
import concurrent.futures
from datetime import datetime
from datetime import timedelta
from cascadelib3.metadata_helper import MetaHelper
from cascadelib3.firestore_helper import FirestoreHelper
import sys
sys.path.insert(0, '../../../cascadelib/cascadelib3')
from cascadelib3.account import Account
import threading

class Auditor():
    def __init__(self, log_level='Info'):
        self.__init_logger(log_level)
        self.sucess = []
        self.issues = []

    def __init_logger(self, log_level='Info'):
        assert (log_level.upper() in ['DEBUG', 'TRACE', 'INFO', 'WARN', 'WARNING', 'ERROR']), AssertionError(" Scope must be one of: {} recieved: {}".format(['all'] + all, scope))
        self.logger = logging.getLogger(sys.argv[0])
        self.logger.setLevel(log_level.upper())
        stream_handler = logging.StreamHandler(sys.stdout)
        file_handler = logging.FileHandler(datetime.now().strftime('{}-%Y-%m-%d_%H-%M-%S.log'.format(sys.argv[0])))
        stream_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
        file_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
        self.logger.addHandler(stream_handler)
        self.logger.addHandler(file_handler)
        self.logger.propagate = False\

    def get_account(self, account_id='', linked=True, target_role_arn=None, profile=None, region=None):
        """
        """
        try:
            account_obj = Account(account_id, linked=linked, region=region)
        except AttributeError as err:
            if str(err) == "'NoneType' object has no attribute 'client'":
                raise ChildProcessError("downstream exception encountered: cascadelib3.get_linked_session: {}: ({})".format(err,
                    "likely assume role permission issue")) from err
        except Exception as err:
            raise (err) 
        assert (account_obj.is_valid()), AssertionError("Unable to assume role")
        return account_obj, account_obj.session

    def get_accounts(self, scope='all', meta=True):
        '''
        uses firestore to gather aws accounts
        ignores inactive accounts
        '''
        accounts = []
        all = ['nonprod', 'pcinonprod', 'prod', 'pciprod']
        assert (scope.lower() in ['all'] + all), AssertionError(" Scope must be one of: {} recieved: {}".format(['all'] + all, scope))
        if scope.lower() == 'all':
            scope = all
        else: 
            scope = [scope]
        try:
            accounts_meta = FirestoreHelper().query_r('aws-accounts')['data']
            for env in scope:
                for account in accounts_meta:
                    if account['active'] and account['environment'].lower() == env.lower():
                        if meta:
                            accounts.append(account)
                        else:
                            accounts.append(account.get('accountNumber'))
        except Exception as err:
            raise (err) 
        assert (len(list(accounts)) > 0), AssertionError(" Unable to retrieve accounts from firestore")
        return accounts

    def get_vpc_regions(self, account_id):
        """

        """
        regions = []
        a, s = self.get_account(account_id)
        try:
            for region in ['us-east-1', 'us-east-2', 'us-west-1']:
                ec2 = s.client('ec2', region_name=region)
                filters = [{'Name': 'tag:'+'Name', 'Values':['vpc_*-*-vpc-??']}]
                vpc_meta = ec2.describe_vpcs(Filters=filters).get('Vpcs')
                if vpc_meta:    
                    regions.append(region)
        except Exception as err:
            raise (err) 
        return regions

    def get_subnets(self, account_id, region='us-west-2'):
        """

        """
        subnets = []
        a, s = self.get_account(account_id)
        try:
            ec2 = s.client('ec2', region_name=region)
            name_internal = [{'Name':'tag:Name', 'Values':['subnet_*_internal_az?']}]
            designation_internal = [{'Name':'tag:Designation', 'Values':['internal']}]
            for meta in ec2.describe_subnets(Filters=name_internal)['Subnets']:
                subnet = meta.get('SubnetId')
                if subnet not in subnets:
                    subnets.append(subnet)
            for meta in ec2.describe_subnets(Filters=designation_internal)['Subnets']:
                subnet = meta.get('SubnetId')
                if subnet not in subnets:
                    subnets.append(subnet)
        except Exception as err:
            raise (err)
        return subnets

    def get_policy(self, account_id, policy_name='-TeamManagedPolicy-'):
        """

        """
        a, s = self.get_account(account_id)
        try:
            iam = s.client('iam')
            policies = iam.list_policies(Scope='Local')['Policies']
            for policy_itr in policies:
                if policy_name in policy_itr['PolicyName']:
                    arn = policy_itr['Arn']
                    name = policy_itr['PolicyName']
            policy = iam.get_policy(
                PolicyArn = arn
            )
            policy_version = iam.get_policy_version(
                PolicyArn = arn,
                VersionId = policy['Policy']['DefaultVersionId']
            )
            statement = policy_version['PolicyVersion']['Document']['Statement']
        except Exception as err:
            raise (err)  
        return {'Name': name, 'Arn': arn, 'Statement': statement}

    def get_event(self, account_id, search_key='EventName', value='ConsoleLogin', start=datetime.now() - timedelta(minutes=120), end=datetime.now(), region='us-west-2'):
        """

        """
        a, s = self.get_account(account_id)
        self.logger.info('perameters: AttributeKey: {}, AttributeValue: {}, StartTime: {}, EndTime: {}, region_name: {}'.format(search_key, value, start, end, region))
        try:
            ct = s.client('cloudtrail', region_name=region)
            response = ct.lookup_events(
                LookupAttributes=[
                    {
                        'AttributeKey': search_key,
                        'AttributeValue': value
                    },
                ],
                StartTime=start,
                EndTime=end,
                MaxResults=100,
            )
            self.logger.info('response: {}'.format(response))
        except Exception as err:
            self.logger.error(err)
            raise (err)  
        return response.get('Events')

    def append_policy(self, account_id, region, subnets, name='-TeamManagedPolicy-'):
        """

        """
        a, s = self.get_account(account_id)
        try:
            iam = s.client('iam')
            policies = iam.list_policies(Scope='Local')['Policies']
            for policy_itr in policies:
                if name in policy_itr['PolicyName']:
                    arn = policy_itr['Arn']
            policy = iam.get_policy(
                PolicyArn = arn
            )
            policy_version = iam.get_policy_version(
                PolicyArn = arn,
                VersionId = policy['Policy']['DefaultVersionId']
            )
            statement = policy_version['PolicyVersion']['Document']['Statement']
            for sid in statement:
                if sid['Sid'] == 'DenyASGToNonInternalSubnets':
                    _subnets = list(sid['Condition']['ForAnyValue:StringNotEquals']['autoscaling:VPCZoneIdentifiers'])
                    sid['Condition']['ForAnyValue:StringNotEquals']['autoscaling:VPCZoneIdentifiers'] = _subnets + subnets       
        except Exception as err:
            raise (err)  
        return sid
        

    def get_statementid(self, policy_statement, statement_id='DenyASGToNonInternalSubnets'):
        """

        """
        try:
            for _sid in policy_statement:
                if _sid['Sid'] == statement_id:
                    sid = _sid  
        except Exception as err:
            raise (err)  
        return sid

    def get_statement_keyvalue(self, statement, keys=[]):
        """

        """
        try:
            for _sid in policy_statement:
                if _sid['Sid'] == statement_id:
                    sid = _sid  
        except Exception as err:
            raise (err)  
        return sid 


    def finder(self, account_id):
        """

        """
        try:
            self.logger.info('searching: {}'.format(account_id))
            events = self.get_event(account_id)
            for event in events:
                self.logger.info('event: {}'.format(event))
        except Exception as err:
            raise(err)

    def get_users(self, account_id, region='us-west-2'):
        """

        """
        self.logger.info('perameters: account_id: {}, region_name: {}'.format(account_id, region))
        a, s = self.get_users(account_id)
        self.logger.info('perameters: account_id: {}, region_name: {}'.format(account_id, region))
        try:
            iam = s.client('iam')
            response = iam.list_users()
            self.logger.info('response: {}'.format(response))
        except Exception as err:
            self.logger.error(err)
            raise (err)  
        return response.get('Users')


    # def finder(self, account_id):
    #     """

    #     """
    #     try:
    #         self.logger.info('searching: {}'.format(account_id))
    #         users = self.get_users(account_id)
    #         self.logger.info('iam_users: {}'.format(users))
    #     except Exception as err:
    #         raise(err)


    def audit(self, scope):
        """

        """

        try:
            billing_account, billing_session = self.get_account(linked=False)
            accounts = self.get_accounts(scope=scope, meta=False)
            # accounts = ['xxxxxxxxxxxxxxxx']
            start_time = datetime.now()
            with concurrent.futures.ThreadPoolExecutor(max_workers=15) as executor:
                executor.map(self.finder, accounts)
            duration = (datetime.now() - start_time).seconds
            self.logger.info("Evaluated {} accounts in {} seconds with max_workers={}".format(len(accounts), duration, 15))
        except Exception as err:
            raise(err)

def get_args(argv):
    parser = argparse.ArgumentParser(description='audit resources')
    parser.add_argument('-e', '--env', required=True, dest='scope', choices=['NonProd', 'Prod', 'PCIProd', 'PCINonProd', "All"],
                        help="Environment(s) to update: NonProd, Prod, PCIProd, PCINonProd, All")
    return parser.parse_args()

if __name__ == '__main__':
    argv = (' '.join(sys.argv[1:]))
    args = get_args(argv)
    auditor = Auditor()
    auditor.logger.info('Running with input: {}'.format(argv))
    auditor.audit(args.scope)

else:
    audit = Resource_Auditer()