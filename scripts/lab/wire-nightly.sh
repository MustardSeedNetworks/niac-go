#!/usr/bin/env bash
# Run the wire-level integration suite and record the result.
#
# This runs from a systemd timer on the lab host rather than from a GitHub
# Actions runner. The suite needs root, iproute2 and a machine whose network
# state it may reshape, and the lab host is RFC1918 with outbound-only
# connectivity, so a GitHub-hosted runner cannot reach it. A *self-hosted*
# runner would reach it, but niac-go is public: a self-hosted runner on a public
# repository lets an outside contributor's workflow execute on this machine,
# inside the lab network. A timer has no inbound surface at all.
#
# Reporting degrades gracefully. The exit status is the primary signal — a
# failure leaves the unit in `systemctl --failed` and the transcript in the
# journal. If a token is present the run additionally files a GitHub issue, but
# the absence of one is not an error, so the timer is useful before anyone
# provisions a credential.
#
# Install on a lab host:
#   sudo install -m0755 scripts/lab/wire-nightly.sh /usr/local/bin/niac-wire-nightly.sh
#   sudo install -m0644 deploy/systemd/niac-wire-nightly.{service,timer} /etc/systemd/system/
#   sudo git clone https://github.com/MustardSeedNetworks/niac-go /var/lib/niac-wire/repo
#   sudo systemctl daemon-reload && sudo systemctl enable --now niac-wire-nightly.timer
#
# Environment:
#   NIAC_WIRE_REPO   checkout to test        (default /var/lib/niac-wire/repo)
#   NIAC_WIRE_STATE  where results are kept  (default /var/lib/niac-wire)
#   NIAC_WIRE_TOKEN  file holding a GitHub token with issues:write (optional)
#   NIAC_WIRE_REPO_SLUG  owner/repo to file against (default MustardSeedNetworks/niac-go)
set -uo pipefail

REPO="${NIAC_WIRE_REPO:-/var/lib/niac-wire/repo}"
STATE="${NIAC_WIRE_STATE:-/var/lib/niac-wire}"
TOKEN_FILE="${NIAC_WIRE_TOKEN:-/etc/niac-wire/github-token}"
SLUG="${NIAC_WIRE_REPO_SLUG:-MustardSeedNetworks/niac-go}"

mkdir -p "$STATE"
started="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
log="${STATE}/last-run.log"

# A clone of its own, never a developer's working checkout: this resets to the
# fetched mainline on every run, which would discard whatever branch and
# uncommitted work a session had open there.
if [[ ! -d "$REPO/.git" ]]; then
	printf 'wire-nightly: no dedicated clone at %s; create one with\n' "$REPO" >&2
	printf '  git clone https://github.com/%s %s\n' "$SLUG" "$REPO" >&2
	exit 78 # EX_CONFIG
fi

# Test the merged mainline.
git -C "$REPO" fetch --quiet origin main 2>&1 | tee "$log"
git -C "$REPO" -c advice.detachedHead=false checkout --quiet FETCH_HEAD 2>&1 | tee -a "$log"
commit="$(git -C "$REPO" rev-parse --short HEAD)"

printf 'wire-nightly: %s at %s\n' "$SLUG" "$commit" | tee -a "$log"
start_epoch=$SECONDS
go test -C "$REPO" -tags integration ./internal/wiretest/... -count=1 -v 2>&1 | tee -a "$log"
status="${PIPESTATUS[0]}"
duration=$((SECONDS - start_epoch))

finished="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
result=$([[ "$status" -eq 0 ]] && echo passed || echo failed)

# A history of results is what "three consecutive green runs" is read from.
printf '{"startedAt":"%s","finishedAt":"%s","commit":"%s","result":"%s","exitCode":%d,"durationSeconds":%d}\n' \
	"$started" "$finished" "$commit" "$result" "$status" "$duration" |
	tee "${STATE}/last-run.json" |
	cat >>"${STATE}/history.jsonl"

if [[ "$status" -eq 0 ]]; then
	printf 'wire-nightly: passed in %ss\n' "$duration"
	exit 0
fi

printf 'wire-nightly: FAILED (exit %d) after %ss\n' "$status" "$duration" >&2

# Optional escalation. Never fail the run because reporting failed — the exit
# status below is the suite's, so a broken token cannot mask a broken product.
if [[ -r "$TOKEN_FILE" ]] && command -v gh >/dev/null; then
	# Read the transcript before the reporting command redirects into it;
	# tailing and appending in one pipeline races on the same file.
	tail_text="$(tail -80 "$log")"
	body="$(
		cat <<-REPORT
			The wire integration suite failed on the lab host.

			Commit: ${commit}
			Started: ${started}
			Exit code: ${status}

			Last 80 lines:

			\`\`\`text
			${tail_text}
			\`\`\`
		REPORT
	)"
	if ! GH_TOKEN="$(<"$TOKEN_FILE")" gh issue create \
		--repo "$SLUG" \
		--title "wire-nightly failed on ${commit}" \
		--label ci \
		--body "$body" >>"$log" 2>&1; then
		printf 'wire-nightly: issue reporting failed; see %s\n' "$log" >&2
	fi
fi

exit "$status"
