#!/bin/bash
#
# start computational server, run as: 
#
# sudo -u $USER-NAME compute-start.sh zone-name host-name

# set -e

srv_zone="$1"
srv_name="$2"

if [ -z "$srv_name" ] || [ -z "$srv_zone" ] ;
then
  echo "ERROR: invalid (empty) server name or zone: $srv_name $srv_zone"
  exit 1
fi

gcloud compute instances start $srv_name --zone $srv_zone
status=$?

if [ $status -ne 0 ];
then
  echo "ERROR $status at start of: $srv_name"
  exit $status
fi

# wait until MPI is ready

for i in 1 2 3 4; do

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
