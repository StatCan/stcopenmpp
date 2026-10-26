#!/usr/bin/env bash
#
# Start oms web-service in the current terminal:
#   ulimit -S -s 65536
#   cd $OM_ROOT
#   bin/oms -oms.Listen localhost:${OMS_PORT} -oms.HomeDir models/home -oms.AllowDownload -oms.AllowUpload -oms.AllowMicrodata -oms.LogRequest ...
#
# Environment:
#   OM_ROOT  - openM++ root folder, default is the parent of this script's folder
#   OMS_PORT - oms web-service port to listen, default: 4040
#
# Any extra arguments to this script are passed on to oms.

pause_exit() {
  echo -n "Press Enter to exit..."
  read -r _
  exit "${1:-0}"
}

# large models may require stack limit increase
# only raise the soft limit, never lower it (e.g. when it is already unlimited)
#
stack_want=65536
stack_cur=$(ulimit -S -s)
stack_max=$(ulimit -H -s)

if [ "$stack_cur" != "unlimited" ] && [ "$stack_cur" -lt "$stack_want" ] ;
then
  if [ "$stack_max" != "unlimited" ] && [ "$stack_max" -lt "$stack_want" ] ;
  then
    echo "WARNING: stack hard limit is ${stack_max} KB, less than ${stack_want} KB; large models may fail"
    stack_want=$stack_max
  fi
  ulimit -S -s "$stack_want" || echo "WARNING: FAILED to set: ulimit -S -s $stack_want"
fi

# set openM++ root folder
#
if [ -z "$OM_ROOT" ] ;
then
  OM_ROOT="$(dirname "$(readlink -f "$0")")/.."
fi

cd "$OM_ROOT" 2>/dev/null || { echo "ERROR: openM++ directory does not exist: $OM_ROOT" ; pause_exit 1 ; }
export OM_ROOT="$PWD"

if [ ! -x "bin/oms" ] ;
then
  echo "ERROR: oms executable not found: $OM_ROOT/bin/oms"
  pause_exit 1
fi

mkdir -p log || { echo "ERROR: cannot create log directory: $OM_ROOT/log" ; pause_exit 1 ; }

# allow to use $MODEL_NAME.ini file in UI for model run
#
export OM_CFG_INI_ALLOW=true
export OM_CFG_INI_ANY_KEY=true
export OM_CFG_TYPE_MAX_LEN="${OM_CFG_TYPE_MAX_LEN:-366}"

# start oms web-service
#
OMS_PORT="${OMS_PORT:-4040}"

oms_cmd=(
  ./bin/oms
  -oms.Listen "localhost:${OMS_PORT}"
  -oms.HomeDir models/home
  -oms.AllowDownload
  -oms.AllowUpload
  -oms.AllowMicrodata
  -oms.PidSaveTo "${OM_ROOT}/log/oms.pid.txt"
  -oms.LogRequest
  -OpenM.LogFilePath "${OM_ROOT}/log/oms.log"
  "$@"
)

echo "OM_ROOT=$OM_ROOT"
printf '%q ' "${oms_cmd[@]}"
echo

# on Ctrl+C let oms stop, then continue below instead of killing this script
trap ':' INT

"${oms_cmd[@]}"
status=$?

trap - INT

if [ $status -eq 130 ] ;
then
  echo " oms web-service terminated by Ctrl+C"
elif [ $status -ne 0 ] ;
then
  echo " oms web-service FAILED, exit code: $status, see log: $OM_ROOT/log/oms.log"
fi

echo "."
pause_exit $status