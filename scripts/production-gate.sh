#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

go test ./...
go vet ./...
go test -race ./...

tmpdir="$(mktemp -d)"
echo "load artifacts: $tmpdir"

pids=()
cleanup() {
	for pid in "${pids[@]:-}"; do
		kill "$pid" 2>/dev/null || true
	done
}
trap cleanup EXIT

free_port() {
	python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}

port="$(free_port)"
health_port="$(free_port)"
salt="1234567890abcdef"
cat >"$tmpdir/server.toml" <<EOF
listen = "127.0.0.1:$port"
data_dir = "$tmpdir/data"
workers = 4
max_players = 254
connect_rate = 512
autosave_interval = "0s"
default_generator = "Classic"
server_name = "Solar Gate"
motd = "Production gate"

[auth]
enabled = true
salt = "$salt"

[heartbeat]
enabled = false
public = false

[debug]
pprof_address = "127.0.0.1:$health_port"
pprof_shutdown_timeout = "2s"
EOF

go run ./cmd/solar start --config "$tmpdir/server.toml" >"$tmpdir/server.log" 2>&1 &
pids+=("$!")

for _ in $(seq 1 80); do
	if curl -fsS "http://127.0.0.1:$health_port/healthz" >/dev/null 2>&1; then
		break
	fi
	sleep 0.25
done
curl -fsS "http://127.0.0.1:$health_port/healthz" >"$tmpdir/health0.json"

go run ./cmd/solar loadtest \
	--address "127.0.0.1:$port" \
	--clients "${SOLAR_LOAD_CLIENTS:-254}" \
	--duration "${SOLAR_LOAD_DURATION:-30s}" \
	--scenario mixed \
	--cpe \
	--auth-salt "$salt" | tee "$tmpdir/loadtest.log"

if [ "${SOLAR_SKIP_REAL_CLIENT:-0}" != "1" ]; then
	"$root/scripts/real-classicube-smoke.sh"
fi
