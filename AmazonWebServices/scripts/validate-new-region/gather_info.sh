#!/usr/bin/env bash
# 05/2015
# Used gather host/service details in order validate service capacity health for 
# EBS-stats service platform before launch of new ec2 region

while getopts 'f:h' flag; do
        case "${flag}" in
                f) infile="${OPTARG}" ;;
                h) echo "$0 -f <filename> " >&2
                        exit 1 ;;
                *) error "Unexpected option ${flag}" >&2
                        echo "$0 -f <filename> " >&2
                        exit 1 ;;
        esac
done

if [ ! -d "logs" ]; then
        mkdir logs
else
        rm -rf logs
        mkdir logs
fi

while IFS=' ' read -r env hc; do
        echo Environment: $env
        if [ "$hc" != "" ]; then
                if [ "$(/apollo/env/envImprovement/bin/expand-hostclass $hc -hosts-only)" != "" ]; then
                        for hn in $(/apollo/env/envImprovement/bin/expand-hostclass $hc -hosts-only); do
                                read state < <(/apollo/env/ApolloCommandLine/bin/apolloHostControlCLI --print $hn 2>&1 | grep currently | awk '{print $4}')
                                echo -e '\t'Hostname: $hn'\t'Status: $state
                                echo -e $env $hc $hn $state >> logs/out.file
                        done
                else
                        echo -e '\t'No host in Hostclass: $hc
                        echo -e $env $hc no-hosts no-state >> logs/out.file
                fi
        else
                echo -e '\t'No Hostclass specified
                echo -e $env no-hclass no-hosts no-state >> logs/out.file
        fi
        echo
done < <(cat $infile | sort -k2)

clear

while read -r env; do
        while read hc hn state; do
                echo $env $hc $hn $state
        done < <(grep $env logs/out.file | awk '{print $2, $3, $4}')
        echo
done < <(cat logs/out.file | awk '{print $1}' | sort -u)
