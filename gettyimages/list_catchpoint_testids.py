#!/usr/bin/env python
# 11/2018
# Authenticates and queries to catchpoint api and returns a all stored test ids group by test type

import argparse
import sys
import numpy
import modules.catchpoint as catchpoint

def get_test_info(credentials, catchpoint_client, sort_key): #gathers all test info for authenticated consumer
    try:
        tests = []
        test_types = []
        for test in catchpoint_client.tests(credentials)['items']:
            test_type = test['type']
            tests.append(test)
            test_types.append(test_type[sort_key])
    except Exception as error:
        print error
        exit
    sort_keys = numpy.unique(numpy.array(test_types))
    return {'tests':tests, 'sort_keys':sort_keys,} #returns dict of all test data and dict of only unique sort_keys

def group_results(tests, sort_keys, sort_key): #groups results by sort_key (name or id)
    grouped_results = []
    for key in sort_keys:
        tests_by_key = []
        for test in tests:
            test_type = test['type']
            if test_type[sort_key] == key:
                test_dict = {'id':test['id'], 'name':test['name'], 'type_name':test_type['name'], 'type_id':test_type['id']}
                tests_by_key.append(test_dict)
        key_dict = {key:tests_by_key}
        grouped_results.append(key_dict)
    return grouped_results #returns array of sort_key dicts containing of array of grouped test dicts 

def main(argv):
    args = get_args(argv)
    version = args.version
    key = args.key
    secret = args.secret
    sort_key = args.sort_key
    uri = 'ui/api/v{}'.format(version)
    credentials = {'client_id':key, 'client_secret':secret}

    catchpoint_client = catchpoint.Catchpoint(api_uri=uri)

    test_data = get_test_info(credentials, catchpoint_client, sort_key)
    tests = test_data['tests']
    sort_keys = test_data['sort_keys']
    
    for groups in group_results(tests, sort_keys, sort_key): #array / dict parsing magic
        for key_group in groups:
            test_ids = []
            sys.stdout.write('test_type: [{}]\n'.format(key_group))
            for test_details in groups[key_group]:
                test_ids.append(test_details['id'])
            sys.stdout.write('test_ids: {}\n\n'.format(test_ids))
    sys.stdout.flush()

#functionalized arg parser to addargparse string check to unit test (placeholder for unit test)
def get_args(argv):
    parser = argparse.ArgumentParser(description='calls catchpoint api returning sorted array of test_ids grouped by type')
    parser.add_argument('-k', '--key', nargs='?', dest='key', help=' <catchpoint REST_API KEY>')
    parser.add_argument('-s', '--secret', nargs='?', dest='secret', help=' <catchpoint REST_API SECRET>')
    parser.add_argument('-v', '--version', nargs='?', default='1', dest='version', help=' <url version "1" or "2">')
    parser.add_argument('-sk', '--sort-key', nargs='?', default='name', dest='sort_key', help='<sort by test_type - "id" or "name">')
    return parser.parse_args()
        
if __name__ == '__main__': #main and argument parsing magic returns: "test_type: [name]  test_ids: [test_id(s), ..]"
    main(sys.argv[1:])
