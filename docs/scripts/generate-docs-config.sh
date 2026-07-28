#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

include_head=false
if [[ ${1:-} == --include-head ]]; then
  include_head=true
fi

first_docs_major=8
release_refs=$(gh release list --exclude-drafts --exclude-pre-releases --limit 1000 --json tagName \
  --jq "[.[].tagName | select((capture(\"^v?(?<major>[0-9]+)[.]0[.]0$\")?.major | tonumber) >= $first_docs_major)][:5][]")

jq --arg release_refs "$release_refs" --argjson include_head "$include_head" '
  .versions = [
    (if $include_head then {label: "Latest", ref: "HEAD", banner: "unreleased"} else empty end),
    ($release_refs | split("\n")[] | select(length > 0) | {ref: .})
  ]
' docs-config.json >docs-config.json.tmp
mv docs-config.json.tmp docs-config.json
