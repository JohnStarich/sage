#!/usr/bin/env bash

tmpfile=$(mktemp)

function do_exit() {
    rm -f "$tmpfile"
}

trap do_exit EXIT

go test ./... -cover -coverprofile "$tmpfile" >&2
go tool cover -func "$tmpfile" | tail -1 | awk '{print $3}'
