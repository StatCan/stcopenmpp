#!/usr/bin/env bash
#
# Start openM++ UI:
#   cd $OM_ROOT
#     # start oms in a new terminal window:
#   x-terminal-emulator -e bin/start_oms.sh
#     # wait until oms is listening, then open UI in browser:
#   xdg-open http://localhost:4040
#
# Environment:
#   OM_ROOT       - openM++ root folder, default is the parent of this script's folder
#   OMS_PORT      - oms web-service port to listen, default: 4040
#   OM_X_TERMINAL - x-terminal to use (may include arguments), default: first found of
#                   x-terminal-emulator, xfce4-terminal, gnome-terminal, konsole, qterminal, ptyxis
#   OM_X_OPEN_URL - command to open URL, default: xdg-open or firefox
#
# Exit codes: 0 = OK, 1 = setup error, 2 = port in use, 3 = oms did not start, 7 = browser failed

self=$(basename "$0")
sd=$(dirname "$(readlink -f "$0")")

OMPP_UI_SH_LOG=""

pause_exit() {
  echo -n "Press Enter to exit..."
  read -r _
  exit "${1:-0}"
}

log() {
  if [ -n "$OMPP_UI_SH_LOG" ] ; then
    echo "$*" | tee -a "$OMPP_UI_SH_LOG"
  else
    echo "$*"
  fi
}

fail() {   # usage: fail exit_code message...
  local code=$1
  shift
  log "ERROR: $*"
  pause_exit "$code"
}

# set openM++ root folder
#
[ -z "$OM_ROOT" ] && OM_ROOT="${sd}/.."

cd "$OM_ROOT" 2>/dev/null || fail 1 "openM++ directory does not exist: $OM_ROOT"
export OM_ROOT="$PWD"

echo "OM_ROOT  = $OM_ROOT"

# check if OM_ROOT is openM++ root
#
if [ ! -x "bin/oms" ] || [ ! -x "bin/start_oms.sh" ] || [ ! -d "html" ] || [ ! -d "models/bin" ] ;
then
  fail 1 "openM++ UI not found at: $OM_ROOT"
fi

mkdir -p log || fail 1 "cannot create log directory: $OM_ROOT/log"

# log configuration
#
OMPP_UI_SH_LOG="${OM_ROOT}/log/${self}.log"

echo "Log:       $OMPP_UI_SH_LOG"
echo "OM_ROOT  = $OM_ROOT" >"$OMPP_UI_SH_LOG"

# which x-terminal to use
#
if [ -z "$OM_X_TERMINAL" ] ;
then
  for t in x-terminal-emulator xfce4-terminal gnome-terminal konsole qterminal ptyxis ; do
    if command -v "$t" >/dev/null 2>&1 ; then
      OM_X_TERMINAL="$t"
      break
    fi
  done
fi
[ -n "$OM_X_TERMINAL" ] || \
  fail 1 "no terminal found (tried: x-terminal-emulator, xfce4-terminal, gnome-terminal, konsole, qterminal, ptyxis); set OM_X_TERMINAL"

# how to open URL: xdg-open or firefox
#
if [ -z "$OM_X_OPEN_URL" ] ;
then
  for t in xdg-open firefox ; do
    if command -v "$t" >/dev/null 2>&1 ; then
      OM_X_OPEN_URL="$t"
      break
    fi
  done
fi
[ -n "$OM_X_OPEN_URL" ] || fail 1 "no URL opener found (tried: xdg-open, firefox); set OM_X_OPEN_URL"

# set oms port to listen
#
[ -z "$OMS_PORT" ] && OMS_PORT=4040
export OMS_PORT
log "OMS_PORT = $OMS_PORT"

# port check uses ss (iproute2) if available, otherwise lsof; if neither, the check is skipped
#
port_tool=""
command -v ss   >/dev/null 2>&1 && port_tool="ss"
[ -z "$port_tool" ] && command -v lsof >/dev/null 2>&1 && port_tool="lsof"

port_listening() {
  case "$port_tool" in
    ss)   ss -Htln "sport = :$1" 2>/dev/null | grep -q . ;;
    lsof) lsof -P -n -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1 ;;
    *)    return 1 ;;
  esac
}

if [ -z "$port_tool" ] ;
then
  log "WARNING: neither ss nor lsof found, cannot check if port $OMS_PORT is in use"
elif port_listening "$OMS_PORT" ;
then
  log "FAILED to start oms web-service, port in use: OMS_PORT=$OMS_PORT"
  log "If port $OMS_PORT used by oms web-service then stop it first"
  pause_exit 2
fi

# start oms web-service in a new terminal window
#
read -r -a term <<< "$OM_X_TERMINAL"     # allow OM_X_TERMINAL to include arguments

case "$(basename "${term[0]}")" in
  gnome-terminal|ptyxis) term_cmd=("${term[@]}" -- ./bin/start_oms.sh) ;;   # -e is deprecated here
  *)                     term_cmd=("${term[@]}" -e ./bin/start_oms.sh) ;;
esac

log "${term_cmd[*]} &"

# setsid detaches the terminal so it keeps running after this script exits
setsid "${term_cmd[@]}" >>"$OMPP_UI_SH_LOG" 2>&1 </dev/null &

# wait for oms to start listening (catches start_oms.sh failures, unlike checking $? of a background job)
#
if [ -n "$port_tool" ] ;
then
  log "Waiting for oms on port $OMS_PORT..."
  for _ in $(seq 1 20) ; do
    port_listening "$OMS_PORT" && break
    sleep 0.5
  done
  port_listening "$OMS_PORT" || \
    fail 3 "oms did not start listening on port $OMS_PORT within 10 seconds, see logs in: $OM_ROOT/log"
else
  sleep 2
fi

# start browser and open UI
#
OMS_URL="http://localhost:${OMS_PORT}"

log "Open openM++ UI in browser:"

read -r -a opener <<< "$OM_X_OPEN_URL"

if [ "$(basename "${opener[0]}")" = "firefox" ] ;
then
  log "${opener[*]} ${OMS_URL} &"
  "${opener[@]}" "$OMS_URL" >>"$OMPP_UI_SH_LOG" 2>&1 &
else
  log "${opener[*]} ${OMS_URL}"
  "${opener[@]}" "$OMS_URL" >>"$OMPP_UI_SH_LOG" 2>&1 || fail 7 "FAILED to open browser at ${OMS_URL}"
fi
# known issue: xdg-open may return 0 when it is not able to start browser

log "Done."
pause_exit 0