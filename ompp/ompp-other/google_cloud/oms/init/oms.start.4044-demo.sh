#!/bin/sh

om_root=$OMPP_ROOT/demo

OM_ROOT=$om_root \
OM_CFG_LOGOUT_URL="/login?logout=true" \
$OMPP_ROOT/bin/oms -l localhost:4044 -oms.RootDir $om_root -ini $OMPP_ROOT/oms.ini >&2 & \
oms_pid=$! \
status=$?
