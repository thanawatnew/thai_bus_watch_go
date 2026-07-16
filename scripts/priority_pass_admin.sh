#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

usage() {
  echo "Usage:"
  echo "  $0 generate [COUNT=50] [FILE=priority-passes.json]"
  echo "  $0 check PASS [FILE=priority-passes.json]"
  echo "  $0 status [URL=http://149.118.52.77]"
}

run_go() {
  if command -v go >/dev/null 2>&1; then
    (cd "$root" && go run ./tools/genpasses "$@")
  else
    docker run --rm -v "$root:/src" -w /src golang:1.25-alpine \
      go run ./tools/genpasses "$@"
  fi
}

command="${1:-}"
case "$command" in
  generate)
    count="${2:-50}"
    file="${3:-priority-passes.json}"
    "$root/scripts/generate_priority_passes.sh" "$count" "$file"
    ;;
  check)
    pass="${2:-}"
    file="${3:-priority-passes.json}"
    if [ -z "$pass" ]; then
      usage
      exit 2
    fi
    run_go -check "$pass" -file "$file"
    ;;
  status)
    url="${2:-http://149.118.52.77}"
    response=$(curl -fsS --max-time 10 "${url%/}/api/access/status")
    if command -v jq >/dev/null 2>&1; then
      printf '%s\n' "$response" | jq '{enabled, active, maxUsers, authorized, rank}'
    else
      printf '%s\n' "$response"
    fi
    ;;
  *)
    usage
    exit 2
    ;;
esac
