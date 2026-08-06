#!/usr/bin/env bash
# Run one presentation pack through the NIAC side of the Link-Live acceptance
# loop: generate it the way the product does (through the API the UI calls),
# start it on its own trunk VLAN, and print the exact CyberScope profile and
# Link-Live upload metadata to use.
#
# The generated YAML is saved and is the authored truth the comparator reads,
# so the thing under test and the thing compared against are the same artifact.
# A hand-edited config is an oracle, never the product — do not substitute one.
#
# Usage:
#   scripts/lab/acceptance.sh <pack-id> [vlan]
#   scripts/lab/acceptance.sh --compare <pack-id> [unit-mac]
#
# Environment:
#   NIAC_URL        daemon base URL           (default https://10.44.40.22:8445)
#   NIAC_API_TOKEN  daemon API token          (required for a non-loopback bind)
#   LAB_OUT         where configs/reports go  (default ./.lab)
set -euo pipefail

NIAC_URL="${NIAC_URL:-https://10.44.40.22:8445}"
LAB_OUT="${LAB_OUT:-.lab}"
CURL=(curl -sk --fail-with-body)
[[ -n "${NIAC_API_TOKEN:-}" ]] && CURL+=(-H "Authorization: Bearer ${NIAC_API_TOKEN}")

# Physical VLAN per pack. Deployment identity — deliberately NOT inside the
# portable pack, per the concurrent-VLAN plan.
vlan_for() {
	case "$1" in
	hospital) echo 200 ;;
	warehouse) echo 201 ;;
	manufacturing) echo 202 ;;
	campus) echo 203 ;;
	retail) echo 204 ;;
	service-provider) echo 205 ;;
	enterprise-scale) echo 299 ;;
	*) return 1 ;;
	esac
}

die() {
	printf '\033[31merror:\033[0m %s\n' "$1" >&2
	exit 1
}

pack_request() {
	"${CURL[@]}" "${NIAC_URL}/api/v1/scenario/packs" |
		python3 -c "
import json,sys
pack_id=sys.argv[1]
for pack in json.load(sys.stdin):
    if pack['id']==pack_id:
        json.dump(pack['request'],sys.stdout); sys.exit(0)
sys.exit('unknown pack '+pack_id)
" "$1"
}

generate() {
	local pack="$1" out="$2"
	pack_request "$pack" >"${out}/${pack}.request.json"
	"${CURL[@]}" -X POST "${NIAC_URL}/api/v1/scenario/generate" \
		-H 'Content-Type: application/json' \
		--data-binary "@${out}/${pack}.request.json" \
		>"${out}/${pack}.generated.json"
	python3 -c "
import json,sys
doc=json.load(open(sys.argv[1]))
open(sys.argv[2],'w').write(doc['content'])
print(json.dumps(doc['manifest']))
" "${out}/${pack}.generated.json" "${out}/${pack}.yaml"
}

start_session() {
	local pack="$1" vlan="$2" out="$3"
	python3 -c "
import json,sys
print(json.dumps({
  'sessionId': sys.argv[1],
  'attachmentMode': 'trunk',
  'accessVlan': int(sys.argv[2]),
  'interface': sys.argv[3],
  'configData': open(sys.argv[4]).read(),
}))
" "$pack" "$vlan" "${LAB_IFACE:-eth0}" "${out}/${pack}.yaml" >"${out}/${pack}.start.json"

	"${CURL[@]}" -X POST "${NIAC_URL}/api/v1/simulation/preflight" \
		-H 'Content-Type: application/json' \
		--data-binary "@${out}/${pack}.start.json" >"${out}/${pack}.preflight.json" ||
		die "preflight failed; see ${out}/${pack}.preflight.json"

	"${CURL[@]}" -X POST "${NIAC_URL}/api/v1/simulation" \
		-H 'Content-Type: application/json' \
		--data-binary "@${out}/${pack}.start.json" >"${out}/${pack}.started.json" ||
		die "start failed; see ${out}/${pack}.started.json"
}

version() { "${CURL[@]}" "${NIAC_URL}/__version" | python3 -c "import json,sys;print(json.load(sys.stdin)['version'])"; }

if [[ "${1:-}" == "--compare" ]]; then
	pack="${2:-}"
	[[ -n "$pack" ]] || die "usage: $0 --compare <pack-id> [unit-mac]"
	[[ -f "${LAB_OUT}/${pack}.yaml" ]] || die "no ${LAB_OUT}/${pack}.yaml — run without --compare first"
	args=(-config "${LAB_OUT}/${pack}.yaml" -latest)
	[[ -n "${3:-}" ]] && args+=(-unit-mac "$3")
	echo "comparing ${pack} against the latest ready discovery..."
	go run ./tools/linklive-acceptance "${args[@]}" | tee "${LAB_OUT}/${pack}.report.json"
	exit "${PIPESTATUS[0]}"
fi

pack="${1:-}"
[[ -n "$pack" ]] || die "usage: $0 <pack-id> [vlan]"
vlan="${2:-$(vlan_for "$pack")}" || die "unknown pack $pack"
mkdir -p "$LAB_OUT"

ver="$(version)"
echo "generating ${pack} through the product API (${NIAC_URL})"
manifest="$(generate "$pack" "$LAB_OUT")"
echo "  manifest: ${manifest}"
echo "starting session ${pack} on physical VLAN ${vlan}"
start_session "$pack" "$vlan" "$LAB_OUT"

today="$(date -u +%Y-%m-%d)"
devices="$(python3 -c "import json;print(json.loads('''${manifest}''')['deviceCount'])")"
links="$(python3 -c "import json;print(json.loads('''${manifest}''')['linkCount'])")"

cat <<EOF

  session is up. On the CyberScope:

    1. AutoTest profile          VLAN ${vlan}
    2. Discovery                 clear results FIRST, then run to 100%
                                 (stale inventory reuses the previous run's
                                  vendor/OUI cadence and corrupts the compare)
    3. Upload To Link-Live       upper-right of Discovery
       If VLAN ${vlan} reports buffered / Link-Live unreachable, leave it
       queued and run an outbound profile (EZ Wired) to flush. Only after
       Discovery is complete.

  Apply this metadata in Link-Live immediately after upload:

    Name     NIAC ${pack} | VLAN ${vlan} | ${ver} | ${today}
    Comment  UI-generated NIAC ${pack} pack: ${devices} authored devices,
             ${links} authored links. NIAC ${ver}.
    Tags     NIAC, ${pack}, VLAN-${vlan}, ${ver}, CyberScope, Acceptance

  Then compare:

    $0 --compare ${pack} [unit-mac]

EOF
