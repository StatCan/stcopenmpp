#!/bin/sh

PATH="/sbin:/bin:/usr/sbin:/usr/bin:$PATH"

OMSD_ROOT=/mirror/data/oms      # must be correct in order to use omsd.ini
OMSD_RUN_DIR=$OMSD_ROOT/run

[ -f "$OMSD_ROOT/omsd.ini" ] && . "$OMSD_ROOT/omsd.ini" # override above settings

[ -d "$OMSD_RUN_DIR" ] || exit 0 # exit OK if no oms pid directory

# read oms pid and terminate processes
#
for f in "$OMSD_RUN_DIR"/oms.*.pid; do

  echo "    Stop $f" >&2;

  oms_pid=$( cat "$f" 2>/dev/null )

  if [ $? -eq 0 ] && [ $oms_pid -gt 1 ];
  then
    kill -TERM $oms_pid >&2
    [ $? -eq 0 ]; rm "$f" >&2
  fi

done

exit 0
