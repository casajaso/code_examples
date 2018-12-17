#!/usr/bin/env bash
# 05/2015
# Used parse gathered host/service details in order validate service capacity health for 
# EBS-stats service platform before launch of new ec2 region

while getopts 'dhe:' flag; do
        case "${flag}" in
                e) target_env="${OPTARG}" ;;
		d) depth="true";;
                h) echo "$0 -e <env_name> " >&2
                        exit 1 ;;
                *) error "Unexpected option ${flag}" >&2
                        echo "$0 -e <env_name> " >&2
                        exit 1 ;;
        esac
done

if [[ ("$depth" = "true") && ("$target_env" != "") ]]; then
	while read -r env; do
		echo $env 
		while read hc hn state; do
			if [ "$state" = "Active" ]; then
				echo -e [1m$hc'\t'$hn'\t'$state[0m
				echo [1mChecking Apollo/Env logs for errors: [0m
				ssh -A -q -n -o ConnectTimeOut=5 -o StrictHostKeyChecking=no $hn 'find /apollo/var/logs/ /apollo/env/*/var/output/logs/ -mtime 1 ! -name "*.gz" | while read fn; do if [ "$(cat $fn | grep -Ei error\|fault\|fail\|creds $fn | grep -Ev default\|error_count=0)" != "" ]; then echo $fn; cat $fn | grep -Ei error\|fault\|fail\|creds $fn | grep -Ev default\|error_count=0; fi; done'
				if [ "$(echo $hn | grep amazon.com)" = "" ]; then
					echo [1mChecking for MaterialSets: [0m
					ssh -A -q -n -o ConnectTimeOut=5 -o StrictHostKeyChecking=no $hn 'sudo logbash --change-id na:region-buildout -c "ls /var/lib/midgard/materials"'
					rgn=$(echo $hn | grep -o lck5'[0-2]')
					case "${rgn}" in
						lck50) opman="ebs-opmanager.lck50-1.ec2.substrate" ;;
                                        	lck51) opman="ebs-opmanager.lck51-1.ec2.substrate" ;;
                                        	lck52) opman="ebs-opmanager.lck52-1.ec2.substrate" ;;
					esac
					ip=$(ssh -A -q -n -o ConnectTimeOut=5 -o StrictHostKeyChecking=no $hn 'hostname -i')
					echo [1mChecking DFDD:[0m
					ssh -A -q -n -o ConnectTimeOut=5 -o StrictHostKeyChecking=no $opman "ebs dfdd getall | grep $ip"
#				else
#					uncomment "else" above and insert deterministic prod host check
				fi
				echo 
			elif [ "$state" = "Provisioning" ]; then
				echo -e [1m$hc'\t'$hn'\t'$state[0m
                        	echo [1mChecking recent Apollo/System log entries: [0m
                        	ssh -A -n -o ConnectTimeOut=5 -o StrictHostKeyChecking=no $hn 'find /apollo/var/logs/ /var/log/ -mtime 1 ! -name "*.gz" | while read fn; do if [ "$(cat $fn | grep -Ei error\|fault\|fail\|creds $fn | grep -Ev default\|error_count=0)" != "" ]; then echo $fn; cat $fn | grep -Ei error\|fault\|fail\|creds $fn | grep -v error_count=0; fi; done'
                        	echo
			else
				echo -e [1m$hc'\t'$hn'\t'$state[0m
				echo
			fi
		done < <(grep $env logs/out.file | awk '{print $2, $3, $4}')
	done < <(cat logs/out.file | grep $target_env | awk '{print $1}' | sort -u)
elif [[ ("$depth" = "true") && ("$target_env" = "") ]]; then
	echo "-e <environment> must be specified when passing -d option"
	exit
else
	if [ "$target_env" != "" ]; then
		while read -r env; do
        		echo $env 
			while read hc hn state; do
				echo -e $hc'\t'$hn'\t'$state
			done < <(grep $env logs/out.file | awk '{print $2, $3, $4}')
			echo
        	done < <(cat logs/out.file | grep $target_env | awk '{print $1}' | sort -u) 
	else
	 	while read -r env; do
                        echo $env 
                        while read hc hn state; do
				echo -e $hc'\t'$hn'\t'$state
                        done < <(grep $env logs/out.file | awk '{print $2, $3, $4}')
			echo
                done < <(cat logs/out.file | awk '{print $1}' | sort -u)
	fi
fi
