#!/usr/bin/env bash
set -euo pipefail

examples_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
output_directory="$examples_directory/bin"
mkdir -p "$output_directory"

cd "$examples_directory"
executable_suffix="$(go env GOEXE)"

printf 'Building topo-plugin-memory\n'
go build -o "$output_directory/topo-plugin-memory$executable_suffix" ./topo-plugin-memory

printf 'Building topo-plugin-ai-portal\n'
go build -o "$output_directory/topo-plugin-ai-portal$executable_suffix" ./topo-plugin-ai-portal

printf 'Plugins built in %s\n' "$output_directory"
