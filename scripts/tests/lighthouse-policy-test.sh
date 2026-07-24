#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
config="$repo_root/.lighthouserc.json"
workflow="$repo_root/.github/workflows/ci.yml"

node -e '
const config = require(process.argv[1]);
const assertions = config.ci.assert.assertions;

if ("categories:seo" in assertions) {
  throw new Error("private NIAC UI must not carry a crawlability-based SEO assertion");
}

for (const category of [
  "categories:performance",
  "categories:accessibility",
  "categories:best-practices",
]) {
  if (assertions[category]?.[0] !== "error") {
    throw new Error(category + " must remain release-blocking");
  }
}
' "$config"

if ! grep -Fq "              - '.lighthouserc.json'" "$workflow"; then
  echo ".lighthouserc.json changes must schedule the frontend and Lighthouse jobs"
  exit 1
fi

echo "Lighthouse policy regression tests passed"
