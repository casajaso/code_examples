#!/usr/bin/python

# 11/2016
# Regex hostname conversion utility, Converts lagacy .border .internal addresses to .substrate and vice-versa

import sys
import re

def border2sub(host):
    hostname = re.sub(r'\.\w+\.ec2\.border\.?$', '', host)
    hostname, az = re.match(r'^(.*)-(\w+\d+-\d+|z-\d+)$', hostname).groups()
    if az.startswith('z-'):
        suffix = 'aes0.internal'
    else:
        suffix = 'ec2.substrate'
    print '{}.{}.{}'.format(hostname, az, suffix)

def sub2border(host):
    suffix = 'ec2.border'
    if 'aes0.internal' in host:
        region = 'iad'
        hostname = re.sub(r'\.aes0\.internal\.?$', '', host)
        hostname, az = hostname.split('.')
    else:
        hostname = re.sub(r'\.ec2\.substrate\.?$', '', host)
        hostname, az = hostname.split('.')
        region = re.sub(r'\d+\-\d+$', '', az)
    print '{}-{}.{}.{}'.format(hostname, az, region, suffix)

def main():
    if '.border' in host:
        border2sub(host)
    elif '.substrate' in host or '.internal' in host:
        sub2border(host)
    else:
        print host

if __name__ == '__main__':
    host = sys.argv[1]
    main()
