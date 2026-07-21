#!/usr/bin/env bash
set -euo pipefail

readonly source_url="https://standards-oui.ieee.org/oui/oui.txt"
readonly output_path="internal/oui/data/oui.txt"

curl --fail --location --retry 3 --output "${output_path}" "${source_url}"

printf 'Updated %s from %s\n' "${output_path}" "${source_url}"
shasum -a 256 "${output_path}"
wc -c "${output_path}"
