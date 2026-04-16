#!/bin/bash
#
# stop Azure server, run as: 
#
# sudo -u $USER-NAME az-stop.sh resource-group host-name

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

# Azure VM stop

for i in 1 2 3 4; do

  az vm deallocate -g "$res_group" -n "$srv_name"

  if [ $status -eq 0 ] ; then break; fi

  sleep 10
done

if [ $status -ne 0 ];
then
  echo "ERROR $status at stop of: $srv_name"
  exit $status
fi

echo "Stop OK: $srv_name"
