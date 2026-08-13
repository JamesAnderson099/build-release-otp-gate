#!/bin/sh
set -eu

go test ./...
go build -o /tmp/build-release-otp-gate ./cmd/release-gate
echo "verified: decision tests passed and single binary built"
