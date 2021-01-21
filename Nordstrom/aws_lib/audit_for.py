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


class PCLib():
    def __init__(self, log_level='Info'):
        self.__init_logger(log_level)
        self.successful = [] # to append successful itterations when looping through itterables
        self.unsucessful = [] # to append un-successful itterations when looping through itterables

    def __init_logger(self, log_level='Info'):
        assert (log_level.upper() in ['DEBUG', 'TRACE', 'INFO', 'WARN', 'WARNING', 'ERROR']), AssertionError(
            "Log Level must be one of: {} recieved: ({})".format(log_level)
        )
        try:
            _filename = datetime.now().strftime('{}-%Y-%m-%d_%H-%M-%S.log'.format(sys.argv[0].split('.')[1]))
            _dir = 'logs'
            if not Path(_dir).exists():
                Path(_dir).mkdir(parents=True)
            log_file = '{}/{}'.format(_dir, _filename)
            self.logger = logging.getLogger(sys.argv[0])
            self.logger.setLevel(log_level.upper())
            stream_handler = logging.StreamHandler(sys.stdout)
            file_handler = logging.FileHandler(log_file)
            stream_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
            file_handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s %(message)s'))
            self.logger.addHandler(stream_handler)
            self.logger.addHandler(file_handler)
            self.logger.propagate = False
        except Exception as err:
            raise (err)

    def load_dict_from_json_file(self, filename):
        """
        """
        try:
            with open(filename, "r") as input:
                data = input.read()
        except Exception as err:
            raise(err) 
        return json.loads(data)       

    def load_json_from_s3_object(self, session, bucket, filename):
        """
        """
        try:
            s3 = session.client('s3')
            response = s3.get_object(Bucket=bucket, Key=filename)
            data = response['Body'].read().decode('utf-8')
        except Exception as err:
            raise(err) 
        return data

    def write_dict_to_csv_file(self, input, filename='{}'.format(sys.argv[0].split('.')[1])):
        """
        """
        self.logger.debug('params: (file_name={})'.format(filename))
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

    def write_dict_to_json_file(self, input, filename='{}'.format(sys.argv[0].split('.')[1])):
        """
        """
        self.logger.debug('params: (file_name={})'.format(filename))
        try:
            _filename = '{}.json'.format(filename)
            _dir = 'results'
            if not Path(_dir).exists():
                Path(_dir).mkdir(parents=True)
            json_file = open('{}/{}'.format(_dir, _filename), 'w')
            json_file.write(self.jsonify(input))
            json_file.close()
        except Exception as err:
            raise(err) 

    def get_aws_session_local(self, profile='nordstrom-federated', region='us-west-2'):
        """
        """
        self.logger.debug('boto3 session_params: (region_name={}, profile_name={})'.format(region, profile))
        try:
            session = boto3.Session(region_name=region, profile_name=profile)
            sts = session.client('sts')
            id = sts.get_caller_identity()
            id.pop('ResponseMetadata')
            self.logger.debug('authenticated identity: ({})'.format(id))
        except Exception as err:
            raise(err) 
        return session

    def get_aws_session_cascadelib(self, account_id=None, linked=True, target_role_arn=None, profile=None, region='us-west-2'):
        """
        """
        self.logger.debug('boto3 session_params: (account_id={}, linked={}, target_role_arn={}, profile={}, region={})'.format(
            account_id, 
            linked, 
            target_role_arn, 
            profile, 
            region)
        )
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
            self.logger.debug('authenticated: ({})'.format(id['Arn']))
        except AttributeError as err:
            if str(err) == "'NoneType' object has no attribute 'client'":
                raise ChildProcessError("downstream exception encountered: cascadelib3.get_linked_session: {}: ({})".format(
                    err, "likely assume role permission issue")
                ) from err
        except Exception as err:
            raise (err) 
        return account

    def get_accounts(self, scope='all', metadata=False): 
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

    def paginate_results(self, client, operation, config={}):
        """
        """
        try:
            assert client.can_paginate(operation), Exception(" operation does not support pagination: {}".format(operation))
            paginator = client.get_paginator(operation)
            response = paginator.paginate(PaginationConfig=config).build_full_result()
        except Exception as err:
            raise(err)          
        return response

    def get_iam_user_details(self, session, get_metadata=False):
        """
        """
        try:
            users = []
            iam = session.client('iam')
            if get_metadata:
                users = self.paginate_results(iam, 'get_account_authorization_details')['UserDetailList']
            else:
                users = self.paginate_results(iam, 'list_users')['Users']
        except Exception as err:
            raise(err)     
        return users

    def get_iam_role_details(self, session, get_metadata=False):
        """
        """
        try:
            roles = []
            iam = session.client('iam')
            if get_metadata:
                roles = self.paginate_results(iam, 'get_account_authorization_details')['RoleDetailList']
            else:
                roles = self.paginate_results(iam, 'list_roles')['Roles']
        except Exception as err:
            raise(err)     
        return roles

    def get_iam_policy_details(self, session, get_metadata=False):
        """
        """
        try:
            policies = []
            iam = session.client('iam')
            if get_metadata:
                policies = self.paginate_results(iam, 'get_account_authorization_details')['Policies']
            else:
                policies = self.paginate_results(iam, 'list_policies')['Policies']
        except Exception as err:
            raise(err)     
        return policies

    def get_iam_group_details(self, session, get_metadata=False):
        """
        """
        try:
            groupss = []
            iam = session.client('iam')
            if get_metadata:
                groupss = self.paginate_results(iam, 'get_account_authorization_details')['GroupDetailList']
            else:
                groupss = self.paginate_results(iam, 'list_groups')['Groups']
        except Exception as err:
            raise(err)     
        return groupss

    def find_flow_log(self, session, vpcid, region, log_destination):
        """
        """
        ec2 = session.client('ec2', region_name=region)
        flowlogs = ec2.describe_flow_logs().get('FlowLogs')
        if not flowlogs:
            return False
        for flowlog in flowlogs:
            if (flowlog.get('ResourceId') == vpcid) and (flowlog.get('LogDestination') == log_destination):
                response = str({"vpc_flowlog": dict(flowlog)})
                if flowlog.get('DeliverLogsStatus') != 'SUCCESS' or flowlog.get('FlowLogStatus') != 'ACTIVE':
                    raise(ValueError(" unhealthy vpc_flowlog definition found {}".format(response)))
                self.logger.warning(" an active vpc_flowlog is already defined: {}".format(response))
                return response
        return False     

    def create_flow_log(self, session, vpcid, region, log_destination, dryrun=False):
        """
        """
        ec2 = session.client('ec2', region_name=region)
        response = ec2.create_flow_logs(
        DryRun=dryrun,
        ResourceIds=[
                    vpcid,
        ],
        ResourceType='VPC',
        TrafficType='ALL',
        LogDestinationType='s3',
        LogDestination=log_destination,
        LogFormat='${account-id} ${interface-id} ${srcaddr} \
                ${dstaddr} ${srcport} ${dstport} ${protocol} \
                ${packets} ${bytes} ${start} ${end} ${action} \
                ${log-status} ${instance-id} ${pkt-dstaddr} \
                ${pkt-srcaddr} ${subnet-id} ${tcp-flags}',
        MaxAggregationInterval=600
        )
        unsuccessful = response.get('Unsuccessful')
        assert (len(unsuccessful) == 0), AssertionError("Unable to configure vpc_flowlog: {}".format(unsuccessful))
        return response

    def jsonify(self, data):
        """
        """
        try:
            json_data = json.dumps(data, indent=4, sort_keys=True, default=str)
        except Exception as err:
            raise(err)
        return json_data

    def whoami(self, session):
        try:
            sts = session.client('sts')
            response = sts.get_caller_identity()
            response.pop('ResponseMetadata')
        except Exception as err:
            raise(err)       
        self.logger.debug(self.jsonify(response))
        return response

    def audit(self, account_id):
        """

        """
        try:
            if self.args.single:
                session = self.base_session
            else:
                account = self.get_aws_session_cascadelib(account_id=account_id, linked=self.use_linked_payer)
                session = account.session
            self.logger.debug('searching: {}'.format(account_id))
            roles = self.get_iam_role_details(session)
            for _role in roles:
                if 'DevUsers' in _role['RoleName']:
                    self.logger.info(('AccountId: {}, HasDevUsers: {}, RoleName: {}'.format(account_id, True, _role['RoleName'])))
                    return True
        except Exception as err:
            raise(err)
        self.logger.info(('AccountId: {}, HasDevUsers: {}, RoleName: {}'.format(account_id, False, None)))
        self.results['No_DevUsers'].append({'AccountId': account_id})
        return False

    def audit_threader(self, accounts):
        """

        """

        try:
            start_time = datetime.now()
            with concurrent.futures.ThreadPoolExecutor(max_workers=15) as executor:
                executor.map(self.audit, accounts)
            duration = (datetime.now() - start_time).seconds
            self.logger.info("Evaluated {} accounts in {} seconds with max_workers={}".format(len(accounts), duration, 15))
        except Exception as err:
            raise(err)

def main(args):
    pclib = PCLib(args.log_level)
    logger = pclib.logger
    pclib.results = {}
    pclib.results['No_DevUsers'] = []
    try:
        pclib.args = args
        logger.debug('args: {}'.format(pclib.args))
        if args.profile:
            pclib.base_session = pclib.get_aws_session_local(args.profile)
            pclib.use_linked_payer = False # use fff001-CascadeBillingRole
        else:
            account = pclib.get_aws_session_cascadelib(linked=False)
            pclib.base_session = account.session
            pclib.use_linked_payer = True # LinkedPayerAccountRole
        if pclib.args.single:
            assert (pclib.args.scope == ''), AssertionError("Cannot specify scope when using single")
            id = pclib.whoami(pclib.base_session)
            accounts = [id['Account']]
        else:
            accounts = pclib.get_accounts(scope=args.scope, metadata=False)
        pclib.audit_threader(accounts)
        pclib.write_dict_to_json_file(pclib.results)
        pclib.write_dict_to_csv_file(pclib.results['No_DevUsers'])
        pclib.logger.debug('recorded ({}) entries'.format(len(pclib.results)))
        logger.info(pclib.jsonify(pclib.results))
        # print(pclib.jsonify(pclib.load_dict_from_json_file('results{}.json'.format(sys.argv[0].split('.')[1])))) # print results JSON
    except Exception as err:
        logger.error(err, exc_info=True)
        os._exit(1)

def get_args(argv):
    parser = argparse.ArgumentParser(description='do a thing')
    parser.add_argument('-p', '--profile', required=False, dest='profile', default=None, help="use AWS named profile")
    parser.add_argument('-f', '--filename', required=False, dest='filename', help="file to read input from")
    parser.add_argument('-l', '--loglevel', required=False, dest='log_level', default='INFO', help="set logging level")
    parser.add_argument('-e', '--env', required=False, dest='scope', default='', choices=['NonProd', 'Prod', 'PCIProd', 'PCINonProd', "All"],
                        help="Environment(s) to update: NonProd, Prod, PCIProd, PCINonProd, All")
    parser.add_argument('-s', '--single', dest='single', action='store_true', help="single account execution")

    return parser.parse_args()

if __name__ == '__main__':
    argv = (' '.join(sys.argv[1:]))
    args = get_args(argv)
    main(args)