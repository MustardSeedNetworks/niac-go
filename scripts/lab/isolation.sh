#!/usr/bin/env bash
# Proves the six presentation packs are isolated: on each pack's VLAN its own
# devices answer SNMP and no other pack's do. Run on pvm01.
#
# Concurrency is only real if the VLANs cannot see each other. Every pack shares
# the same gateway address, so a leak would be silent — a device answering from
# the wrong VLAN looks exactly like a device that is simply there.
set -u

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

sysname() { snmpget -v2c -c NetAllyDemo -t 2 -r 0 -Ovq "$1" 1.3.6.1.2.1.1.5.0 2>/dev/null; }

# A sub-interface left over from an earlier probe keeps its own route to a pack's
# space, so every VLAN under test reaches that pack through the stale link and
# the check reports a leak that is not there. Start from a clean slate.
for stale in $(ip -br link show | awk '$1 ~ /^vmbr0\./ {sub(/@.*/, "", $1); print $1}'); do
	ip link del "$stale"
done

failures=0
for pack in "${!VLAN[@]}"; do
	vlan=${VLAN[$pack]}
	link=vmbr0.$vlan
	ip link del "$link" 2>/dev/null
	ip link add link vmbr0 name "$link" type vlan id "$vlan" || continue
	ip addr add 10.254.200.240/24 dev "$link"
	ip link set "$link" up
	for target in "${!ROUTE[@]}"; do
		ip route add "${ROUTE[$target]}" via 10.254.200.1 dev "$link" 2>/dev/null
	done
	sleep 2

	own=$(sysname "${PROBE[$pack]}")
	if [[ -z $own ]]; then
		printf 'FAIL vlan %s: %s did not answer on its own VLAN\n' "$vlan" "$pack"
		failures=$((failures + 1))
	else
		printf 'vlan %-4s %-17s own device answers as %s\n' "$vlan" "$pack" "$own"
	fi

	for other in "${!PROBE[@]}"; do
		[[ $other == "$pack" ]] && continue
		leaked=$(sysname "${PROBE[$other]}")
		if [[ -n $leaked ]]; then
			printf 'FAIL vlan %s: %s answered from %s space as %s\n' \
				"$vlan" "$pack" "$other" "$leaked"
			failures=$((failures + 1))
		fi
	done
	ip link del "$link"
done

printf '\nisolation failures: %d\n' "$failures"
exit $((failures > 0))
