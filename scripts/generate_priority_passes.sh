#!/bin/sh
set -eu

count="${1:-50}"
output="${2:-priority-passes.json}"
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
umask 077

if command -v go >/dev/null 2>&1; then
  (cd "$root" && go run ./tools/genpasses -count "$count") > "$output"
else
  docker run --rm -v "$root:/src" -w /src golang:1.25-alpine \
    go run ./tools/genpasses -count "$count" > "$output"
fi

echo "Generated $count ranked priority passes in $output"
