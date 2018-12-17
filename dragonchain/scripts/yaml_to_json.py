#!/usr/bin/env python
# 09/2018
# code snipits for converting yaml to json used to deploy cloudformation stack through boto3

import os
import sys
import io
import json
import yaml

yaml_file = '/Users/jason/Documents/work_root/scripts/DRGN/MultiAcctEphm/initialPermissions.yml'
json_file = '/Users/jason/Documents/work_root/scripts/DRGN/MultiAcctEphm/initialPermissions.json'
with io.open(yaml_file, 'r', encoding='utf8') as yaml_cft:
    data = yaml.load(yaml_cft.read())
    with io.open(json_file, 'w', encoding='utf8') as json_cft:
        str_ = json.dumps(data, sort_keys=True, indent=4, ensure_ascii=False, encoding='utf8')
        json_cft.write(unicode(str_))

print yaml.dump(yaml.load(json.dumps(json.loads(open(temp).read()))), allow_unicode=True, default_flow_style=False)

json_file = '/Users/jason/Documents/work_root/git/repos/Dragonchain_inc/hopper-api/scripts/ephemeral/sub-account-ephemeral/cft.json'
yaml_file = '/Users/jason/Documents/work_root/git/repos/Dragonchain_inc/hopper-api/scripts/ephemeral/sub-account-ephemeral/cft.yaml'
with io.open(json_file, 'r', encoding='utf8') as json_cft:
    data = yaml.dump(yaml.load(json_cft.read()), encoding=None, default_flow_style=False)
    with io.open(yaml_file, 'w', encoding='utf8') as yaml_cft:
        yaml_cft.write(unicode(data))
