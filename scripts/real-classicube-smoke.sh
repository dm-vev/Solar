#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cc_bin="$root/third_party/ClassiCube/ClassiCube"
default_zip="$root/third_party/ClassiCube/misc/n64/files/default.zip"
classicube_zip="$root/third_party/ClassiCube/misc/dreamcast/classicube.zip"

command -v Xvfb >/dev/null || { echo "Xvfb is required" >&2; exit 1; }
[ -x "$cc_bin" ] || { echo "missing executable $cc_bin" >&2; exit 1; }
[ -f "$default_zip" ] || { echo "missing $default_zip" >&2; exit 1; }
[ -f "$classicube_zip" ] || { echo "missing $classicube_zip" >&2; exit 1; }

tmpdir="$(mktemp -d)"
echo "artifacts: $tmpdir"

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
workers = 2
max_players = 8
connect_rate = 16
autosave_interval = "0s"
default_generator = "Classic"
server_name = "Solar Real CC"
motd = "Real client auth"

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

(cd "$root" && go run ./cmd/solar start --config "$tmpdir/server.toml" >"$tmpdir/server.log" 2>&1) &
pids+=("$!")

for _ in $(seq 1 80); do
	if curl -fsS "http://127.0.0.1:$health_port/healthz" >/dev/null 2>&1; then
		break
	fi
	sleep 0.25
done
curl -fsS "http://127.0.0.1:$health_port/healthz" >"$tmpdir/health0.json"

prepare_client() {
	local dir="$1"
	mkdir -p "$dir/texpacks"
	cp "$cc_bin" "$dir/"
	cp "$default_zip" "$dir/texpacks/default.zip"
	cp "$classicube_zip" "$dir/texpacks/classicube.zip"
	cat >"$dir/options.txt" <<'EOF'
mode-classic=false
nostalgia-customblocks=true
nostalgia-usecpe=true
launcher-updates=false
show-emptyservers=true
fps-limit=60fps
EOF
}

for dir in c1 c2 bad; do
	prepare_client "$tmpdir/$dir"
done

display=""
for candidate in $(seq 520 620); do
	if Xvfb ":$candidate" -screen 0 1280x800x24 -nolisten tcp >"$tmpdir/xvfb.log" 2>&1 & then
		xvfb_pid="$!"
		sleep 0.3
		if kill -0 "$xvfb_pid" 2>/dev/null; then
			display=":$candidate"
			pids+=("$xvfb_pid")
			break
		fi
	fi
done
[ -n "$display" ] || { echo "failed to start Xvfb" >&2; exit 1; }

mppass() {
	printf '%s%s' "$salt" "$1" | md5sum | awk '{print $1}'
}

(cd "$tmpdir/c1" && DISPLAY="$display" LIBGL_ALWAYS_SOFTWARE=1 ./ClassiCube realcc1 "$(mppass realcc1)" 127.0.0.1 "$port" >"$tmpdir/c1.out" 2>&1) &
pids+=("$!")
sleep 0.8
(cd "$tmpdir/c2" && DISPLAY="$display" LIBGL_ALWAYS_SOFTWARE=1 ./ClassiCube realcc2 "$(mppass realcc2)" 127.0.0.1 "$port" >"$tmpdir/c2.out" 2>&1) &
pids+=("$!")

join_ok=0
for i in $(seq 1 35); do
	curl -fsS "http://127.0.0.1:$health_port/healthz" >"$tmpdir/health-join-$i.json" || true
	if grep -q '"players":2' "$tmpdir/health-join-$i.json"; then
		join_ok=1
		break
	fi
	sleep 1
done
[ "$join_ok" = 1 ] || { echo "real clients did not both join" >&2; tail -n 80 "$tmpdir/server.log" >&2; exit 1; }

if command -v import >/dev/null; then
	DISPLAY="$display" import -window root "$tmpdir/two-clients.png" 2>/dev/null || true
fi

(cd "$tmpdir/bad" && DISPLAY="$display" LIBGL_ALWAYS_SOFTWARE=1 ./ClassiCube realbad 00000000000000000000000000000000 127.0.0.1 "$port" >"$tmpdir/bad.out" 2>&1) &
pids+=("$!")

for i in $(seq 1 10); do
	curl -fsS "http://127.0.0.1:$health_port/healthz" >"$tmpdir/health-bad-$i.json" || true
	if grep -q '"players":3' "$tmpdir/health-bad-$i.json"; then
		echo "invalid mppass client joined" >&2
		exit 1
	fi
	sleep 1
done

curl -fsS "http://127.0.0.1:$health_port/healthz"
echo
