#!/usr/bin/env bash

tmpfile=$(mktemp)

# Automatically clean up, even on early exit
function do_exit() {
    rm -f "$tmpfile"
}

trap do_exit EXIT

go test ./... -race -cover -coverprofile "$tmpfile" >&2
coverage=$(go tool cover -func "$tmpfile" | tail -1 | awk '{print $3}')
echo '##########################' >&2
printf '### Coverage is %6s ###\n' "$coverage" >&2
echo '##########################' >&2
echo "$coverage"
