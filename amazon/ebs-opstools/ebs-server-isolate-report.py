# !/apollo/bin/env /apollo/env/EBSPython/bin/python

# 12/2017
# service ran on a daily cadence to track a silent-fail edge case
# that had gone un-noticed for 500+ days. preventing over 2000 storage servers 
# from completing isolation/build workflows and re-joining the storage fleet
#
# Finds EBS storage servers stuck in isolate workflow and determine
# root cause through boolean logic; reults are grouped zonaly to regional dynamodb

import argparse
import bender.ebs.location as locator
import boto3

from bender.ebs.esds import ESDS
from bender.ebs.oracle import Oracle
from bender.locator import get_service_endpoint
from botocore.config import Config as boto_config
from datetime import datetime
from pyodinhttp import odin_retrieve_pair
from retrying import retry


def get_location_info():
    """This module creates a dictionary containing regional and zonal
        information.
    Example:
        get_region_info()
    Args:
        None
    Returns:
        dict: {'internal_region':'str', 'public_region':'str',
               'availability_zone':'str', 'odin_stack':'str'}
    """
    internal_region = locator.get_internal_region()
    public_region = locator.get_public_region()
    availability_zone = locator.get_ebs_zone()
    odin_stack = locator.region_odin_stack(internal_region)
    return {'internal_region':internal_region,
            'public_region':public_region,
            'availability_zone':availability_zone,
            'odin_stack':odin_stack}


def get_credentials(odin_stack):
    """This module determines the odin endpoint and creates dictionary
        contianing aws creds for ebs-server-isolate user in the ebs-ops-metrics
        account.
    Example:
        get_credentials(odin_stack='default')
    Args:
        str: (region='str')
    Returns:
        dict: {'credential':str, 'principal':str}
    """
    odin_material = 'com.amazon.ebs.ebs-server-isolate-report.{}'.format(odin_stack)
    principal,credential = odin_retrieve_pair(odin_material)
    return {'principal':principal.data,
            'credential':credential.data}


def get_epoch():
    """This module returns the time in seconds since epoch Thursday, 1 January
        1970.
    Example:
        get_epoch()
    Args:
        None
    Returns:
        int: Seconds from EPOCH
    """
    time_now = datetime.utcnow()
    return int(time_now.strftime("%s"))


def get_utc():
    """This module returns the current UTC in 24hr notation.
    Example:
        get_utc()
    Args:
        None
    Returns:
        int: YYYYMMDDHHMMSS
    """
    time_now = datetime.utcnow()
    return int(time_now.strftime("%Y%m%d%H%M%S"))


def get_isolating_hosts():
    """This module creates a list of dictionaries for ebs-servers in isolate
        state.
    Example:
        gen_asset_ids()
    Args:
        None
    Returns:
        list: of dict(s): [{'ip_address':str, 'is_default_pool':bool,
                            'days_in_isolate':int}]
    """
    esds = ESDS()
    ebs_servers = []
    for ebs_server in esds.get_server_aspect_for_all_servers('state'):
        if ebs_server.state_name == 'Isolating':
            ip_address = ebs_server.server_id.ip_address
            is_default_pool = ebs_server.pool_default
            epoch_now = get_epoch()
            days_in_isolate = (int(epoch_now * 1000
                               - int(ebs_server.state_entry_time))
                               / 86400000)
            server_info = {'ip_address':ip_address,
                           'is_default_pool':is_default_pool,
                           'days_in_isolate':days_in_isolate}
            ebs_servers.append(server_info)
    return ebs_servers


@retry(wait_exponential_multiplier=1000, wait_exponential_max=10000, stop_max_delay=30000)
def get_volume_info(ip_address):
    """This module creates a list of dictionaries with volume host details for
        provided ip_address.
    Example:
        get_volumes_oracle(ip_address='22.113.131.254')
    Args:
        str: (ip_address='str')
    Returns:
        list: of dict: [{'volume_id':str, 'is_master':bool,
                         'is_peer_isolating':bool, 'is_solo_master':bool}]
    """
    oracle = Oracle()
    volumes = []
    for volume in oracle.list_volumes_on_host(ip_address):
        is_peer_isolating = bool()
        is_solo_master = bool()
        volume_id = volume['ebs-volume-id']
        slave_ip_address = oracle.get_blessed_pair(volume_id)['ebsSlave']
        master_ip_address = oracle.get_blessed_pair(volume_id)['ebsMaster']
        if volume['is-master']:
            peer_ip_address = slave_ip_address
        else:
            peer_ip_address = master_ip_address
            is_peer_isolating = get_peer_state(peer_ip_address)
        if slave_ip_address == master_ip_address:
            is_solo_master = True
        volume_info = {'volume_id':volume_id,
                       'is_master':volume['is-master'],
                       'is_peer_isolating':is_peer_isolating,
                       'is_solo_master':is_solo_master}
        volumes.append(volume_info)
    return volumes


def get_peer_state(ip_address):
    """This module determines if volume peer host is also in isolate state.
    Example:
        get_peer_state(ip_address='22.113.131.254')
    Args:
        str: (ip_address='str')
    Returns:
        bool: is_peer_isolating=bool
    """
    esds = ESDS()
    is_peer_isolating = bool()
    try:
        state = (esds.get_server_aspect(ip_address, 'state').state_name)
    except Exception: # hosts checked into ESDS but ESDS cannot determine state
        is_peer_isolating = 'unknown'
    if state == 'Isolating':
        is_peer_isolating = True
    return is_peer_isolating


def set_dynamodb_client(internal_region, public_region, principal, credential):
    """This module establishes a boto3 dynamodb client connection
    Example:
        set_dynamodb_client(internal_region='iad', public_region='us-east-1',
                            principal='<aws_access_key_id>',
                            credential='<aws_credential>'):
    Args:
        str: (internal_region='str', public_region='str', principal='str',
              credential='str')
    Returns:
        boto3 client: dynamodb
    """
    dynamodb_endpoint = 'https://' + get_service_endpoint(service_name='dynamodb',
                                                          region=internal_region)
    dynamodb_client = boto3.client('dynamodb',
                      region_name=public_region,
                      aws_access_key_id=principal,
                      aws_secret_access_key=credential,
                      endpoint_url=dynamodb_endpoint,
                      config=boto_config(signature_version='s3v4'))
    return dynamodb_client


def set_dynamodb_table(internal_region, public_region, table_name, principal,
                       credential):
    """This module establishes a boto3 dynamodb resource connection
    Example:
        set_dynamodb_client_connect(internal_region='iad',
                                    public_region='us-east-1',
                                    table_name='ebs-server-isolate-report-
                                    <internal region>',
                                    principal='<aws_access_key_id>',
                                    credential='<aws_credential>'):
    Args:
        str: (internal_region='str', public_region='str', table_name='str',
              principal='str', credential='str')
    Returns:
        boto3 resource: table
    """
    dynamodb_endpoint = 'https://' + get_service_endpoint(service_name='dynamodb',
                                                          region=internal_region)
    dynamodb_resource = boto3.resource('dynamodb',
                        region_name=public_region,
                        aws_access_key_id=principal,
                        aws_secret_access_key=credential,
                        endpoint_url=dynamodb_endpoint,
                        config=boto_config(signature_version='s3v4'))
    dynamodb_table = dynamodb_resource.Table(table_name)
    return dynamodb_table


def dynamodb_create_table(internal_region, public_region, table_name, principal,
                          credential):
    """This module checks/creates dynamodb table artifact
    Example:
        dynamodb_create_table(internal_region='iad', public_region='us-east-1',
                              table_name='ebs-server-isolate-report-<internal region>',
                              principal='<aws_access_key_id>',
                              credential='<aws_credential>'):
    Args:
        str: (internal_region='str', public_region='str', table_name='str',
              principal='str', credential='str')
    Returns:
        dynamodb table: ebs-server-isolate-report-<internal region>
    """
    dynamodb_client = set_dynamodb_client(internal_region,
                                          public_region,
                                          principal,
                                          credential)
    try:
        dynamodb_client.describe_table(TableName=table_name)
        print 'Found Table: {}'.format(table_name)
    except dynamodb_client.exceptions.ResourceNotFoundException:
        dynamodb_client.create_table(
            TableName = table_name,
            KeySchema = [
                {
                    'AttributeName': 'volume_id',
                    'KeyType': 'HASH'
                }
            ],
            AttributeDefinitions = [
                {
                    'AttributeName': 'volume_id',
                    'AttributeType': 'S'
                }
            ],
            ProvisionedThroughput = {
                'ReadCapacityUnits': 5,
                'WriteCapacityUnits': 5
            }
        )
        dynamodb_client.get_waiter('table_exists').wait(TableName=table_name)
        print 'Created Table: {}'.format(table_name)


def dynamodb_update_table(internal_region, public_region, table_name, principal,
                          credential, volume):
    """This module adds/updates dynamodb table key value artifacts
    Example:
        dynamodb_update_table(internal_region='iad', public_region='us-east-1',
                              table_name='ebs-server-isolate-report-<internal region>',
                              principal='<aws_access_key_id>',
                              credential='<aws_credential>',
                              volume={<volume dict>}):
    Args:
        str: (internal_region='str', public_region='str', table_name='str',
              principal='str', credential='str', volume={dict})
    Returns:
        dynamodb table key values:
    """
    dynamodb_table = set_dynamodb_table(internal_region,
                                        public_region,
                                        table_name,
                                        principal,
                                        credential)
    volume['timestamp'] = get_utc()
    dynamodb_table.put_item(
        Item = {
            'volume_id': volume['volume_id'],
            'availability_zone': volume['availability_zone'],
            'timestamp': volume['timestamp'],
            'ip_address': volume['ip_address'],
            'days_in_isolate': volume['days_in_isolate'],
            'is_master': str(volume['is_master']),
            'is_default_pool': str(volume['is_default_pool']),
            'is_peer_isolating': str(volume['is_peer_isolating']),
            'is_solo_master': str(volume['is_solo_master'])
        }
    )


def post_dynamodb(internal_region, public_region, principal, credential, volumes):
    """This itterates through a list of volume dictionaries and post data to
        dynamodb table
    Example:
        post_dynamodb(internal_region='iad', public_region='us-east-1',
                      principal='<aws_access_key_id>',
                      credential='<aws_credential>', volumes={volume} or
                      [{volume}]):
    Args:
        str: (internal_region='str', public_region='str', principal='str',
              credential='str', volumes={dict} or [{list of dict}])
    Returns:
        dynamodb table: key values
    """
    table_name = 'ebs-server-isolate-report-{}'.format(internal_region)
    dynamodb_create_table(internal_region,
                          public_region,
                          table_name,
                          principal,
                          credential)
    dynamodb_table = set_dynamodb_table(internal_region,
                                        public_region,
                                        table_name,
                                        principal,
                                        credential)
    try:
        pre_count = int(dynamodb_table.scan()['ScannedCount'])
    except: # Table does not exist
        pre_count = 0
    for volume in volumes:
        dynamodb_update_table(internal_region,
                              public_region,
                              table_name,
                              principal,
                              credential,
                              volume)
    new_count = int(dynamodb_table.scan()['ScannedCount'])
    print 'Previous records: {} total: {}'.format(pre_count, new_count)


def main(post_results_to):
    if post_results_to == 'term':
        for ebs_server in get_isolating_hosts():
            print '\nebs-server-ip:{}\tdefault_pool:{}\tdays_in_isol:{}'.format(
                   ebs_server['ip_address'],
                   ebs_server['is_default_pool'],
                   ebs_server['days_in_isolate'])
            for volume in get_volume_info(str(ebs_server['ip_address'])):
                print '\tvol_id:{}\tmst:{}\tsolo_mst:{}\tpeer_isol:{}'.format(
                       volume['volume_id'],
                       volume['is_master'],
                       volume['is_solo_master'],
                       volume['is_peer_isolating'])
    if post_results_to == 'db':
        location = get_location_info()
        credentials = get_credentials(location['odin_stack'])
        volumes = []
        for ebs_server in get_isolating_hosts():
            for volume in get_volume_info(str(ebs_server['ip_address'])):
                volume_info = {'internal_region':location['internal_region'],
                               'availability_zone':location['availability_zone'],
                               'ip_address':ebs_server['ip_address'],
                               'is_default_pool':ebs_server['is_default_pool'],
                               'days_in_isolate':ebs_server['days_in_isolate'],
                               'volume_id':volume['volume_id'],
                               'is_master':volume['is_master'],
                               'is_peer_isolating':volume['is_peer_isolating'],
                               'is_solo_master':volume['is_solo_master']}
                volumes.append(volume_info)
        post_dynamodb(location['internal_region'],
                      location['public_region'],
                      credentials['principal'],
                      credentials['credential'],
                      volumes)


if  __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Find volume information for \
                                     hosts stuck in isolate')
    parser.add_argument('post_results_to', help=' term or db')
    args = parser.parse_args()
    post_results_to = args.post_results_to
    main(post_results_to)
