#!/usr/bin/env python3

import json
import argparse
import sys
import csv

def main(args):
    with open(args.filename, "r") as data:
        _r = []
        events = json.load(data).get('events')
        for event_itr in events:
            message = event_itr.get('message').split(' ')
            if message[8] == '200':
                _r.append(message[2])
    _u = set(_r)
    users = []
    for user in _u:
        users.append({'UserName': user, 'AuthenticationCount': _r.count(user)})
    results = {'AuthenticatedRequests': len(list(_r)), 'UniqueUserNames': len(list(_u)), 'Requests': users}
    print(json.dumps(results, indent=4, sort_keys=False))
    data_file = open('{}.csv'.format(args.filename.split('.')[0]), 'w')
    row = 0
    csv_writer = csv.writer(data_file)
    for user in results['Requests']:
        if row == 0:
            header = user.keys()
            csv_writer.writerow(header) 
            row += 1
        csv_writer.writerow(user.values()) 
    data_file.close()


def get_args(argv):
    parser = argparse.ArgumentParser(description='audit resources')
    parser.add_argument('-f', '--filename', required=True, dest='filename', help="file to read input from")
    return parser.parse_args()

if __name__ == '__main__':
    argv = (' '.join(sys.argv[1:]))
    args = get_args(argv)
    main(args)