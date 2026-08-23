<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# BreachSAFE PDF documentation

This documentation describes the Go PDF compiler, its input contract, the container
workflow, and the evidence boundaries around generated reports.

## Documentation map

| Audience | Page | Purpose |
| --- | --- | --- |
| New operator | [First report](tutorials/first-report.md) | Build the image and render one report |
| Operator | [Container pipeline](how-to/container-pipeline.md) | Connect QuReddy, CBOM, and PDF output |
| Integrator | [Input and output contracts](reference/contracts.md) | Exact files, fields, digests, and limits |
| Maintainer | [Architecture](explanation/architecture.md) | Layer boundaries and trust decisions |
| Contributor | [Release gates](contributors/release-gates.md) | Tests, image checks, and release evidence |

## Contents

- [Product boundary](#product-boundary)
- [Quickstart](quickstart.md)
- [Architecture and data flow](explanation/architecture.md)
- [Security model](explanation/security.md)
- [First report](tutorials/first-report.md)
- [Container pipeline](how-to/container-pipeline.md)
- [Troubleshooting](how-to/troubleshooting.md)
- [CLI reference](reference/cli.md)
- [Go library and call reference](reference/library-api.md)
- [Input and output contracts](reference/contracts.md)
- [RenderResult reference](reference/render-result.md)
- [Release gates](contributors/release-gates.md)

## Product boundary

BreachSAFE PDF is a renderer and admission boundary. It does not collect endpoint
evidence, interpret OSCAL, compose ePacks, sign artifacts, or store runs. Those
responsibilities belong to the caller.

The compiler accepts exact QuReddy scan JSON, an exact CycloneDX 1.7 CBOM, and a
bounded render request. It emits PDF bytes and a machine-readable RenderResult.

## Version status

The report binary exposes `--version`. The image build supplies the generator version
through the `REPORT_VERSION` build argument. External producer versions are read from
the producer command and preserved in the CBOM, scan JSON, request manifest, or
RenderResult. They are not guessed by the renderer.
