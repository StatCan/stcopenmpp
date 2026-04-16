#!/bin/bash
#
# stop computational server, run as: 
#
# sudo -u $USER-NAME compute-stop.sh zone-name host-name

# set -e

srv_zone="$1"
srv_name="$2"

if [ -z "$srv_name" ] || [ -z "$srv_zone" ] ;
then
  echo "ERROR: invalid (empty) server name or zone: $srv_name $srv_zone"
  exit 1
fi

for i in 1 2 3 4 5 6 7; do

  gcloud compute instances stop $srv_name --zone $srv_zone
  status=$?

  if [ $status -eq 0 ] ; then break; fi

  sleep 10
done

if [ $status -ne 0 ];
then
  echo "ERROR $status at stop of: $srv_name"
  exit $status
fi

echo "Stop OK: $srv_name"
