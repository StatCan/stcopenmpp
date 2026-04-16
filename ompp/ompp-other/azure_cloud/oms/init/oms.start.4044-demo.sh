#!/bin/sh

oms_user=demo
oms_port=4044

om_root=$OMPP_ROOT/${oms_user}

OM_ROOT=$om_root \
OM_CFG_LOGOUT_URL="/login?logout=true" \
$OMPP_ROOT/bin/oms -l :${oms_port} -oms.RootDir $om_root -oms.Name ${oms_port}_${oms_user} -ini $OMPP_ROOT/oms.ini >&2 & \
oms_pid=$! \
status=$?
