#!/usr/bin/env bash
# M4-7 duration evidence: hold all six presentation packs for 24 hours and prove
# nothing drifts. Run on pvm01.
#
# Isolation (isolation.sh) proves the packs cannot see each other at one instant.
# This proves they survive time — the failure modes that only appear after hours
# are a leaked goroutine, a growing heap, a silently dead session, and a daemon
# that restarted and took its sessions with it.
#
# Two planes are sampled because either can fail alone. The API plane reports
# what the daemon believes; the wire plane asks each pack's own device over SNMP
# on its own VLAN, which is the only authoritative check that all six sessions
# are still answering — the daemon exposes no all-sessions endpoint, only the
# selected one.
#
# Usage:
#   scripts/lab/soak.sh [duration-seconds] [interval-seconds]
#
# Environment:
#   CTID          simulator container            (default 304)
#   SOAK_OUT      samples + report destination   (default ./.soak)
set -u

DURATION=${1:-86400}
INTERVAL=${2:-900}
CTID=${CTID:-304}
SOAK_OUT=${SOAK_OUT:-.soak}

declare -A VLAN=(
	[hospital]=200 [warehouse]=201 [manufacturing]=202
	[campus]=203 [retail]=204 [service-provider]=205
)
declare -A PROBE=(
	[hospital]=10.51.200.21 [warehouse]=10.61.200.21 [manufacturing]=10.91.200.21
	[campus]=10.71.200.21 [retail]=10.81.200.21 [service-provider]=10.101.200.21
)
declare -A ROUTE=(
	[hospital]=10.51.0.0/16 [warehouse]=10.61.0.0/16 [manufacturing]=10.91.0.0/16
	[campus]=10.71.0.0/16 [retail]=10.81.0.0/16 [service-provider]=10.101.0.0/16
)
PACKS=(hospital warehouse manufacturing campus retail service-provider)

mkdir -p "$SOAK_OUT"
samples=$SOAK_OUT/samples.jsonl
report=$SOAK_OUT/report.txt

# The daemon binds loopback only, so every API sample goes through the container.
# curl is not installed there; python3 is.
api_sample() {
	pct exec "$CTID" -- python3 -c '
import json, ssl, urllib.request
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

def get(path):
    return urllib.request.urlopen("https://127.0.0.1:8445" + path, context=ctx, timeout=10).read().decode()

out = {}
for line in get("/metrics").splitlines():
    if line.startswith("#") or not line.strip():
        continue
    name, _, value = line.partition(" ")
    out[name] = float(value)
out["commit"] = json.loads(get("/__version"))["commit"]
print(json.dumps(out))
' 2>/dev/null
}

# One VLAN sub-interface at a time, exactly as isolation.sh does. Holding all six
# up at once would give six routes to the same gateway address and the answer
# would no longer identify which VLAN it came from.
wire_sample() {
	local answered=0 missing=""
	for pack in "${PACKS[@]}"; do
		local vlan=${VLAN[$pack]} link=vmbr0.${VLAN[$pack]}
		ip link del "$link" 2>/dev/null
		ip link add link vmbr0 name "$link" type vlan id "$vlan" 2>/dev/null || continue
		ip addr add 10.254.200.240/24 dev "$link" 2>/dev/null
		ip link set "$link" up
		# The probe address sits in the pack's own space, not in the transit
		# /24, so it is only reachable through the pack gateway.
		ip route add "${ROUTE[$pack]}" via 10.254.200.1 dev "$link" 2>/dev/null
		sleep 2
		if [[ -n $(snmpget -v2c -c NetAllyDemo -t 2 -r 1 -Ovq "${PROBE[$pack]}" \
			1.3.6.1.2.1.1.5.0 2>/dev/null) ]]; then
			answered=$((answered + 1))
		else
			missing="$missing $pack"
		fi
		ip link del "$link" 2>/dev/null
	done
	printf '%s|%s' "$answered" "${missing# }"
}

# A sub-interface left behind by an interrupted run keeps its own route into a
# pack's space and makes the next probe answer from the wrong VLAN.
for stale in $(ip -br link show | awk '$1 ~ /^vmbr0\./ {sub(/@.*/, "", $1); print $1}'); do
	ip link del "$stale"
done

started=$(date +%s)
deadline=$((started + DURATION))
round=0
failures=0
first_json=""
prev_uptime=0
restarts=0

printf 'soak: %ss total, sampling every %ss, container %s\n' "$DURATION" "$INTERVAL" "$CTID"
printf 'samples -> %s\n\n' "$samples"

while [[ $(date +%s) -lt $deadline ]]; do
	round=$((round + 1))
	now=$(date -u +%Y-%m-%dT%H:%M:%SZ)

	api=$(api_sample)
	IFS='|' read -r answered missing <<<"$(wire_sample)"

	if [[ -z $api ]]; then
		printf '%s round %-4s API UNREACHABLE\n' "$now" "$round"
		failures=$((failures + 1))
		printf '{"time":"%s","round":%d,"apiReachable":false}\n' "$now" "$round" >>"$samples"
		sleep "$INTERVAL"
		continue
	fi

	uptime=$(printf '%s' "$api" | python3 -c 'import json,sys;print(int(json.load(sys.stdin).get("niac_uptime_seconds",0)))')
	goroutines=$(printf '%s' "$api" | python3 -c 'import json,sys;print(int(json.load(sys.stdin).get("niac_goroutines_total",0)))')
	heap=$(printf '%s' "$api" | python3 -c 'import json,sys;print(int(json.load(sys.stdin).get("niac_memory_usage_bytes",0)))')
	errors=$(printf '%s' "$api" | python3 -c 'import json,sys;print(int(json.load(sys.stdin).get("niac_errors_total",0)))')
	drops=$(printf '%s' "$api" | python3 -c 'import json,sys;print(int(json.load(sys.stdin).get("niac_udp_proxy_overload_drops_total",0)))')

	# uptime going backwards is the only reliable restart signal; systemd's
	# Restart=always makes a crash look like a healthy service afterwards.
	if [[ $prev_uptime -gt 0 && $uptime -lt $prev_uptime ]]; then
		printf '%s round %-4s DAEMON RESTARTED (uptime %ss < %ss)\n' "$now" "$round" "$uptime" "$prev_uptime"
		restarts=$((restarts + 1))
		failures=$((failures + 1))
	fi
	prev_uptime=$uptime

	if [[ $answered -ne 6 ]]; then
		printf '%s round %-4s ONLY %s/6 PACKS ANSWERED — missing:%s\n' \
			"$now" "$round" "$answered" " $missing"
		failures=$((failures + 1))
	fi

	rss=$(pct exec "$CTID" -- ps -o rss= -p "$(pct exec "$CTID" -- pgrep -f 'niac daemon' | head -1)" 2>/dev/null | tr -d ' ')

	python3 -c '
import json, sys
doc = json.loads(sys.argv[1])
doc.update({
  "time": sys.argv[2], "round": int(sys.argv[3]), "apiReachable": True,
  "packsAnswering": int(sys.argv[4]), "packsMissing": sys.argv[5].split() or [],
  "rssKb": int(sys.argv[6] or 0),
})
print(json.dumps(doc))
' "$api" "$now" "$round" "$answered" "$missing" "${rss:-0}" >>"$samples"

	[[ -z $first_json ]] && first_json=$api

	printf '%s round %-4s packs %s/6  goroutines %-5s heap %sMB  rss %sMB  errors %s  drops %s  up %sh\n' \
		"$now" "$round" "$answered" "$goroutines" "$((heap / 1048576))" \
		"$((${rss:-0} / 1024))" "$errors" "$drops" "$((uptime / 3600))"

	sleep "$INTERVAL"
done

first_goroutines=$(printf '%s' "$first_json" | python3 -c 'import json,sys;print(int(json.load(sys.stdin).get("niac_goroutines_total",0)))')
first_heap=$(printf '%s' "$first_json" | python3 -c 'import json,sys;print(int(json.load(sys.stdin).get("niac_memory_usage_bytes",0)))')

{
	printf 'NIAC 24-hour soak — M4-7 duration evidence\n\n'
	printf 'started            %s\n' "$(date -u -d "@$started" +%Y-%m-%dT%H:%M:%SZ)"
	printf 'ended              %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	printf 'rounds             %d at %ss\n' "$round" "$INTERVAL"
	printf 'daemon restarts    %d\n' "$restarts"
	printf 'goroutines         %s -> %s\n' "$first_goroutines" "${goroutines:-?}"
	printf 'heap               %sMB -> %sMB\n' "$((first_heap / 1048576))" "$((${heap:-0} / 1048576))"
	printf 'errors / drops     %s / %s\n' "${errors:-?}" "${drops:-?}"
	printf 'failures           %d\n' "$failures"
} | tee "$report"

exit $((failures > 0))
