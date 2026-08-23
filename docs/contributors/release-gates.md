<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Release gates

## Local Go gates

Run from the repository root:

```bash
gofmt -d cmd internal
go vet ./...
go test ./...
GOMAXPROCS=2 go test -p 1 -race ./...
go mod verify
```

The repository gate script also runs the configured security and anti-pattern
checks when their tools are available:

```bash
./scripts/run-gates.sh
```

Record unavailable external tools as `NOT RUN`; do not turn a missing tool into a
passing result.

## Container gates

Build with a pinned base image and a version argument:

```bash
docker build \
  --build-arg REPORT_VERSION=0.1.1 \
  --tag breachsafe-pdf:v0.1.1 \
  .
```

Verify the runtime identity:

```bash
docker run --rm breachsafe-pdf:v0.1.1 --version
```

The runtime must be non-root and must expose only the report compiler entrypoint.

## Release evidence

A release record should include:

- source commit;
- image tag and immutable digest;
- report binary version output;
- Go test and race results;
- dependency verification result;
- representative RenderResult with input and PDF digests;
- PDF metadata and visual render checks;
- explicit unsupported capability results.

The image is not a published release until the registry digest and release record
are verified externally.
