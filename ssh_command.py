#!/usr/bin/env python

#ssh utility that runs specified command on remote host(s)
#supports bastion/jumpbox proxies as well as password, ssh-agent, and ssh identitiy file authentication.

import argparse
import sys
import paramiko
import getpass

def get_args(argv): #gets vars passed via cli
    parser = argparse.ArgumentParser(description='runs commands on remote host over ssh')
    parser.add_argument('-u', '--user', nargs='?', dest='username', help=' <username>')
    parser.add_argument('-n', '--name', nargs='?', dest='hostname', help=' <hostname>')
    parser.add_argument('-p', '--proxy', nargs='?', default=None, dest='proxy', help=' <(OPTIONAL)proxy>')
    parser.add_argument('-i', '--id-file', nargs='?', default=None, dest='identityfile', help=' <(OPTIONAL)identity file(PEM)>')
    parser.add_argument('-c', '--command', nargs='?', dest='command', help=' <\'";" seperated command(s)\'>')
    return parser.parse_args()

def get_opts(identityfile, proxy): #sets logic around optional vars 
    if (proxy != None):
        proxy_command = 'ssh -qxTA {} nc {} {}'.format(proxy, hostname, '22')
        proxy_socket = paramiko.proxy.ProxyCommand(proxy_command)
    else:
        proxy_socket = proxy
    if (identityfile != None):
        passphrase = getpass.getpass(prompt='passphrase for {}: '.format(identityfile))
        keyfile = paramiko.RSAKey.from_private_key_file(identityfile, password=passphrase)
    else:
        keyfile = identityfile
    return {'proxy':proxy_socket, 'keyfile':keyfile}

def run_ssh(hostname, username, command, keyfile, proxy): #creates ssh session and runs command
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        ssh_client.connect(hostname=hostname, username=username, timeout=None, allow_agent=True, look_for_keys=True, pkey=keyfile, sock=proxy)
        stdin, stdout, stderr = ssh_client.exec_command(command) 
        if stdout.channel.recv_exit_status() != 0:
            print '[ERROR][exec_command]: {}'.format(stderr.read())
        print stdout.read()
        ssh_client.close()
    except Exception as err:
        print '[ERROR][ssh_client.connect]: {}'.format(err)

def main(argv):
    args = get_args(argv)
    opts = get_opts(args.identityfile, args.proxy)
    run_ssh(args.hostname, args.username, args.command, opts['keyfile'], opts['proxy'])
        
if __name__ == '__main__': 
    main(sys.argv[1:])
