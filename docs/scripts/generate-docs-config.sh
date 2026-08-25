#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

include_head=false
if [[ ${1:-} == --include-head ]]; then
  include_head=true
fi

first_docs_major=8
docs_major_versions_max=5
release_refs=$(gh release list --exclude-drafts --exclude-pre-releases --limit 1000 --json tagName \
  --jq "
    def major_version: ltrimstr(\"v\") | split(\".\")[0] | tonumber;

    [
      .[].tagName
      | select(test(\"^v?[0-9]+[.][0-9]+[.][0-9]+$\"))
      | select(major_version >= $first_docs_major)
    ]
    | unique_by(major_version)
    | reverse
    | .[:$docs_major_versions_max][]
  ")

jq --arg release_refs "$release_refs" --argjson include_head "$include_head" '
  .versions = [
    (if $include_head then {label: "Latest", ref: "HEAD", banner: "unreleased"} else empty end),
    ($release_refs | split("\n")[] | select(length > 0) | {ref: .})
  ]
' docs-config.json >docs-config.json.tmp
mv docs-config.json.tmp docs-config.json
