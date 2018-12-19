#!/usr/bin/env python

#usage: ssh_command.py [-h] [-c ['COMMAND' '...']] [-b [BASTIAN]] [-u [USERNAME]] [-i [IDENTITYFILE]] [-p] [hostname]
#
#run commands on remote host over ssh
#
#positional arguments:
#  hostname              target hostname or ip (***required)
#
#optional arguments:
#  -h, --help            show this help message and exit
#  -c ['COMMAND' '...'], --command ['COMMAND' '...']
#                        command(s) to execute on target host (***required)
#                        example: -c 'command' '...' OR -c 'command; ...'
#  -b [BASTIAN], --bastian [BASTIAN]
#                        proxy through a bastion/jumpbox (default: None)
#  -u [USERNAME], --username [USERNAME]
#                        username on target host (default: jcasas)
#  -i [IDENTITYFILE], --identityfile [IDENTITYFILE]
#                        use ssh identify file; opens prompt (default: None)
#  -p, --password        use password; opens prompt (default: no flag set)

import argparse
import sys
import paramiko
import getpass

def get_args(argv): #gets vars passed via cli
    parser = argparse.ArgumentParser(description='run commands on remote host over ssh')
    parser.add_argument('hostname', nargs='?', help=' target hostname or ip (***required)')
    parser.add_argument('-c', '--command', required=True, nargs='+', help=' command(s) to execute on target host (***required) example: -c \'command\' \'...\' OR -c \'command; ...\'')
    parser.add_argument('-b', '--bastian', nargs='?', default=None, help=' proxy through a bastion/jumpbox (default: %(default)s)')
    parser.add_argument('-u', '--username', nargs='?', default=(getpass.getuser()), help=' username on target host (default: %(default)s)')
    parser.add_argument('-i', '--identityfile', nargs='?', default=None, help=' use ssh identify file; opens prompt (default: %(default)s)')
    parser.add_argument('-p', '--password', action='store_true', help=' use password; opens prompt (default: no flag set)')
    return parser.parse_args()

def get_opts(password, username, identityfile, bastian, hostname): #sets logic around optional vars
    if (password):
        password = getpass.getpass(prompt='input password for {}: '.format(username))
    else:
        password = None
    if (identityfile != None):
        passphrase = getpass.getpass(prompt='input passphrase for {}: '.format(identityfile))
        keyfile = paramiko.RSAKey.from_private_key_file(identityfile, password=passphrase)
    else:
        keyfile = identityfile
    if (bastian != None):
        proxy_command = 'ssh -qxTA {} nc {} {}'.format(bastian, hostname, '22')
        proxy_socket = paramiko.proxy.ProxyCommand(proxy_command)
    else:
        proxy_socket = bastian
    return {'proxy':proxy_socket, 'keyfile':keyfile, 'password':password}

def run_ssh(hostname, username, password, keyfile, proxy, commands): #creates ssh session and runs command
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        ssh_client.connect(hostname=hostname, username=username, password=password, timeout=None, allow_agent=True, look_for_keys=True, pkey=keyfile, sock=proxy)
        for command in commands:
            stdin, stdout, stderr = ssh_client.exec_command(command)
            if stdout.channel.recv_exit_status() != 0:
                print '[***ERROR][REMOTE][EXEC]: {}'.format(stderr.read())
            print stdout.read()
        ssh_client.close()
    except Exception as err:
        print '[***ERROR][LOCAL][CONNECT]: {}'.format(err)

def main(argv):
    args = get_args(argv)
    opts = get_opts(args.password, args.username, args.identityfile, args.bastian, args.hostname)
    run_ssh(args.hostname, args.username, opts['password'], opts['keyfile'], opts['proxy'], args.command)
        
if __name__ == '__main__': 
    main(' '.join(sys.argv[1:]))
