<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Contributing to BreachSAFE PDF

[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square)](https://github.com/paul007ex/breachsafe-pdf/pulls)
[![CI](https://github.com/paul007ex/breachsafe-pdf/actions/workflows/ci.yml/badge.svg)](https://github.com/paul007ex/breachsafe-pdf/actions/workflows/ci.yml)

## Contents

1. [Before you contribute](#before-you-contribute)
2. [Architecture rules](#architecture-rules)
3. [Quality gates](#quality-gates)
4. [Pull requests](#pull-requests)
5. [Security](#security)
6. [License](#license)

## Before you contribute

Read [`README.md`](README.md), [`docs/explanation/architecture.md`](docs/explanation/architecture.md),
[`docs/reference/contracts.md`](docs/reference/contracts.md), [`AGENTS.md`](AGENTS.md), and
[`CLAUDE.md`](CLAUDE.md). Open an issue before a non-trivial behavior change and keep one
behavior change per PR.

## Architecture rules

- Keep the CLI thin and put workflow sequencing in `internal/evidenceapp`.
- Keep producer parsing and correlation in `internal/admission`.
- Keep the source-neutral model in `internal/evidence`.
- Keep PDF presentation in `internal/pdf`.
- Call FPDF through its Go API. Do not add templates or remote resources.
- Keep `internal/` until a public Go API is intentionally versioned.
- Do not make the compiler execute QuReddy, OpenSSL, shell commands, or arbitrary paths.
- Preserve exact input bytes and fail closed on digest or correlation errors.

## Quality gates

Run `gofmt`, `go vet ./...`, `go test ./...`, the race suite, `go mod verify`, and
`git diff --check`. Run `./scripts/run-gates.sh` when the shared golden toolchain is
available. CI also runs Staticcheck, golangci-lint, govulncheck, OSV scan, and a
no-skipped-test check. A missing local tool is `NOT RUN`, not `PASS`.

For PDF changes, render a fixture and inspect `pdfinfo`, extracted text, and a rendered
page. Keep generated artifacts outside Git tracking.

## Pull requests

Use a focused branch and Conventional Commit subject such as `docs: explain
RenderResult provenance`, `fix(admission): reject uncorrelated CBOM`, or
`test(pdf): cover long posture values`. Describe the behavior, contract or issue,
commands and exit statuses, gates run, and any unverified claim. Do not pass gates by
skipping tests or hiding errors.

## Security

Do not report vulnerabilities in a public issue. Use [`SECURITY.md`](SECURITY.md). Do not
commit secrets, private scan data, tokens, or unredacted endpoint logs.

## License

First-party contributions use `PolyForm-Noncommercial-1.0.0`. Third-party and generated
material keeps its original license and notices. Run `reuse lint` when changing metadata.
