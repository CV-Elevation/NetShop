#!/usr/bin/env bash
set -euo pipefail

# NetShop service ports started by run.sh
PORTS=(50051 50052 50053 50054 50055 50056 50057 8080)
GRACE_SECONDS="${GRACE_SECONDS:-5}"

join_by_comma() {
  local IFS=","
  echo "$*"
}

PORTS_CSV="$(join_by_comma "${PORTS[@]}")"

get_listen_pids() {
  lsof -nP -tiTCP:"$PORTS_CSV" -sTCP:LISTEN 2>/dev/null | sort -u || true
}

print_ports() {
  lsof -nP -iTCP:"$PORTS_CSV" -sTCP:LISTEN 2>/dev/null || true
}

echo "[stop] target ports: ${PORTS_CSV}"

PIDS="$(get_listen_pids)"
if [[ -z "$PIDS" ]]; then
  echo "[stop] no running listeners found"
  exit 0
fi

echo "[stop] sending SIGTERM to: $(echo "$PIDS" | tr '\n' ' ')"
kill -TERM $PIDS

for ((i=1; i<=GRACE_SECONDS; i++)); do
  sleep 1
  REMAINING="$(get_listen_pids)"
  if [[ -z "$REMAINING" ]]; then
    echo "[stop] all services stopped gracefully"
    exit 0
  fi
  echo "[stop] waiting graceful shutdown... ${i}/${GRACE_SECONDS}"
done

REMAINING="$(get_listen_pids)"
if [[ -n "$REMAINING" ]]; then
  echo "[stop] force killing remaining PIDs: $(echo "$REMAINING" | tr '\n' ' ')"
  kill -KILL $REMAINING || true
fi

FINAL="$(get_listen_pids)"
if [[ -z "$FINAL" ]]; then
  echo "[stop] all services stopped"
else
  echo "[stop] some listeners still remain"
  print_ports
  exit 1
fi
