#!/bin/sh
set -e

# Configurable hosts/ports via env (with defaults)
REDIS_HOST=${REDIS_HOST:-redis}
REDIS_PORT=${REDIS_PORT:-6379}
POSTGRES_HOST=${POSTGRES_HOST:-postgres}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
KAFKA_HOST=${KAFKA_HOST:-localhost}
KAFKA_PORT=${KAFKA_PORT:-9094}

# Optional startup tuning
START_DELAY=${START_DELAY:-0}
WAIT_INTERVAL=${WAIT_INTERVAL:-1}
WAIT_TIMEOUT=${WAIT_TIMEOUT:-120}

check_tcp() {
  host="$1"; port="$2"
  python - <<PY
import socket,sys
try:
    s=socket.create_connection(("${host}", ${port}), timeout=1)
    s.close()
    sys.exit(0)
except:
    sys.exit(1)
PY
}

wait_for() {
  host="$1"; port="$2"
  echo "wait: checking ${host}:${port}"
  elapsed=0
  while ! check_tcp "$host" "$port"; do
    if [ "$elapsed" -ge "$WAIT_TIMEOUT" ]; then
      echo "wait: timeout waiting for ${host}:${port}" >&2
      return 1
    fi
    sleep "$WAIT_INTERVAL"
    elapsed=$((elapsed + WAIT_INTERVAL))
  done
  echo "wait: ${host}:${port} is available"
}

if [ "${START_DELAY}" != "0" ]; then
  echo "startup: initial sleep ${START_DELAY}s"
  sleep "${START_DELAY}"
fi

# Wait for dependencies
wait_for "$REDIS_HOST" "$REDIS_PORT" || exit 1
wait_for "$POSTGRES_HOST" "$POSTGRES_PORT" || exit 1
wait_for "$KAFKA_HOST" "$KAFKA_PORT" || exit 1

echo "All dependencies ready, starting service"
exec uvicorn main:app --host 0.0.0.0 --port 8000
