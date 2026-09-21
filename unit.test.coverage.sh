#!/usr/bin/env bash

set -e

# Get to workdir
cd "$(realpath "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

tmpfile="$(mktemp)"

echo "tmpfile: ${tmpfile}"

go test -count=1 -coverprofile="${tmpfile}" ./...

go tool cover -func="${tmpfile}"