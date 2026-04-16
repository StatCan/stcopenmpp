#!/bin/sh

### BEGIN INIT INFO
# Provides:          oms
# Required-Start:    $local_fs $network
# Required-Stop:     $local_fs $network
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: oms web-service for openM++
### END INIT INFO

PATH="/sbin:/bin:/usr/sbin:/usr/bin:$PATH"

OMSD_ROOT=/mirror/data/oms      # must be correct in order to use omsd.ini
OMSD_INIT_DIR=$OMSD_ROOT/init
OMSD_RUN_DIR=$OMSD_ROOT/run

OMPP_ROOT=/mirror/data
# OMS_EXE=$OMPP_ROOT/bin/oms
OMS_USER=oms

[ -f "$OMSD_ROOT/omsd.ini" ] && . "$OMSD_ROOT/omsd.ini"   # override above settings

# [ -x "$OMS_EXE" ] && [ -d "$OMSD_INIT_DIR" ] && [ -d "$OMSD_RUN_DIR" ] || exit 0 # exit OK if oms not found

[ -d "$OMSD_INIT_DIR" ] && [ -d "$OMSD_RUN_DIR" ] || exit 0   # exit OK if oms directories not found

umask 022

# . /etc/rc.d/init.d/functions
# . /lib/lsb/init-functions

start() {
  echo "Start oms" >&2
  su -c "$OMSD_ROOT/omsd-start.sh" "$OMS_USER" >&2
  return $?
}

stop() {
  echo "Stop oms" >&2
  su -c "$OMSD_ROOT/omsd-stop.sh" "$OMS_USER" >&2
  return $?
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    stop
    start
    ;;
  *)
  echo "Usage: $0 {start|stop|restart}" >&2
  exit 1

esac

exit 0
