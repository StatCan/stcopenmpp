#!/bin/sh

PATH="/sbin:/bin:/usr/sbin:/usr/bin:$PATH"

OMSD_ROOT=/mirror/data/oms      # must be correct in order to use omsd.ini
OMSD_INIT_DIR=$OMSD_ROOT/init
OMSD_RUN_DIR=$OMSD_ROOT/run

OMPP_ROOT=/mirror/data
# OMS_EXE=$OMPP_ROOT/bin/oms

[ -f "$OMSD_ROOT/omsd.ini" ] && . "$OMSD_ROOT/omsd.ini" # override above settings

# [ -x "$OMS_EXE" ] && [ -d "$OMSD_INIT_DIR" ] || exit 0 # exit OK if oms not found

[ -d "$OMSD_INIT_DIR" ] || exit 0   # exit OK if oms init dir not found

[ -d "$OMSD_RUN_DIR" ] || OMSD_RUN_DIR=

# large models may require stack limit increase
#
MODELS_STACK_65K=67108864
#

# start oms and save pid
#
for f in "$OMSD_INIT_DIR"/oms.start.*.sh; do
  if [ ! -x "$f" ] ; then continue; fi

  echo "    Start $f" >&2;

  . "$f"

  if [ $status -ne 0 ];
  then
    echo "ERROR $status at source: $f" >&2
    exit 1
  fi

  # set stack 65K for models
  if ! prlimit --stack=$MODELS_STACK_65K: -p $oms_pid >&2;
  then
    echo "ERROR $? at prlimit pid: $oms_pid of: $f" >&2
    exit 1
  fi

  # save oms PID
  [ -n "$OMSD_RUN_DIR" ] && ps $oms_pid 1> /dev/null 2>&1 && echo "$oms_pid" >$OMSD_RUN_DIR/oms.$oms_pid.pid

done

exit 0
