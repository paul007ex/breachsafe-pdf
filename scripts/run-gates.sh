#!/bin/sh
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
set -eu

image="${BREACHSAFE_GOLDEN_GO_IMAGE:-ghcr.io/paul007ex/breachsafe-golden-go:1.26.6}"
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

exec docker run --rm \
  -v "$repo_root:/workspace" \
  -w /workspace \
  "$image" \
  -c '
set -eu
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
go test -race ./...
go mod verify
staticcheck ./...
golangci-lint run
govulncheck ./...
osv-scanner scan source -r .
gosec ./...
'
