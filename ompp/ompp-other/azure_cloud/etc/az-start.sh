#!/bin/bash
#
# start Azure server, run as: 
#
# sudo -u $USER-NAME az-start.sh resource-group host-name

# set -e

res_group="$1"
srv_name="$2"

if [ -z "$srv_name" ] || [ -z "$res_group" ] ;
then
  echo "ERROR: invalid (empty) server name or resource group: $srv_name $res_group"
  exit 1
fi

# login

az login --identity
status=$?

if [ $status -ne 0 ];
then
  echo "ERROR $status from az login at start of: $res_group $srv_name"
  exit $status
fi

# Azure VM start 

az vm start -g "$res_group" -n "$srv_name"
status=$?

if [ $status -ne 0 ];
then
  echo "ERROR $status at: az vm start -g $res_group -n $srv_name"
  exit $status
fi

# wait until MPI is ready

for i in 1 2 3 4 5; do

  sleep 10

  echo "[$i] mpirun -n 1 -H $srv_name hostname"

  mpirun -n 1 -H $srv_name hostname
  status=$?

  if [ $status -eq 0 ] ; then break; fi
done

if [ $status -ne 0 ];
then
  echo "ERROR $status from MPI at start of: $srv_name"
  exit $status
fi

echo "Start OK: $srv_name"
