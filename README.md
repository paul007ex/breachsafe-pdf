<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# BreachSAFE PDF

Reusable Go PDF report compiler for BreachSAFE OSS, Prowler, Enterprise, and evidence-pack
workflows.

The compiler accepts exact scan/CBOM inputs, validates their correlation, projects them into a
source-neutral report model, and emits deterministic PDF bytes plus a machine-readable
`RenderResult`. Evidence collection, OSCAL interpretation, ePack composition, signing, and
storage remain outside this repository.

## Architecture

```text
QuReddy / CBOM / report request
              |
              v
       bounded admission
              |
              v
       validated report model
              |
              v
       internal/pdf renderer
              |
              v
       PDF bytes + RenderResult
```

The implementation uses Codeberg FPDF directly, with fixed design tokens, bounded pagination,
embedded assets, no remote resources, and no arbitrary template execution. The initial command is
`cmd/evidence-report`; a stable library API will be exposed after the input contract is finalized.

## Development

```sh
GOTOOLCHAIN=go1.26.6 CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=go1.26.6 CGO_ENABLED=0 go test -race ./...
go mod verify
```

Generated PDFs, scan runs, ePacks, binaries, and rendered images belong in external versioned run
directories and are not committed to this source repository.
