#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

include_head=false
if [[ ${1:-} == --include-head ]]; then
  include_head=true
fi

release_refs=$(
  for ref in $(gh release list --exclude-drafts --exclude-pre-releases --limit 1000 --json tagName --jq '[.[] | select(.tagName | test("^v?[0-9]+\\.0\\.0$"))][:5][].tagName'); do
    if git cat-file -e "$ref:docs/docs-config.json" 2>/dev/null; then
      echo "$ref"
    fi
  done
)

jq --arg release_refs "$release_refs" --argjson include_head "$include_head" '
  .versions = [
    (if $include_head then {label: "Latest", ref: "HEAD", banner: "unreleased"} else empty end),
    ($release_refs | split("\n")[] | select(length > 0) | {ref: .})
  ]
' docs-config.json >docs-config.json.tmp
mv docs-config.json.tmp docs-config.json
