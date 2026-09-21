#!/usr/bin/env bash

set -e

# Get to workdir
cd "$(realpath "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

version="${VERSION:-"undefined"}"

export CGO_ENABLED=0

mkdir -p ./build

build_binary() {
    local goos="${1:?}"
    local goarch="${2:?}"
    output="./build/notification-gateway_${goos}_${goarch}"

    echo "Building ${output} (${goos}/${goarch})..."
    GOOS="${goos}" GOARCH="${goarch}" go build \
        -o "${output}" \
        -mod vendor \
        -trimpath \
        -ldflags "-s -w -X github.com/ismdeep/notification-gateway/version.Version=${version}" \
        .
}

echo "Starting build (version: ${version})..."

build_binary linux  amd64
build_binary linux  arm64
build_binary darwin amd64
build_binary darwin arm64

echo "Build completed."
