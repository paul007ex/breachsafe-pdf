<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# CLI reference

## Contents

1. [Identity command](#identity-command)
2. [Render command](#render-command)
3. [Profile CLI](#profile-cli)
4. [Streams and exit codes](#streams-and-exit-codes)
5. [Container invocation](#container-invocation)

## Identity command

```bash
breachsafe-pdf --version
```

Output is one line containing the binary name and build version. The version is
embedded at build time and is also written to PDF metadata and RenderResult.

## Render command

```text
breachsafe-pdf render --profile breachsafe/community \
  --request REQUEST.json \
  --cbom CBOM.json \
  --scan-json SCAN.json \
  --pdf REPORT.pdf \
  --result REPORT.result.json
```

| Option | Required | Description |
| --- | --- | --- |
| `--profile` | yes | Versioned report profile identifier |
| `--request` | yes | Bounded render request JSON |
| `--cbom` | yes | Exact CycloneDX 1.7 CBOM file |
| `--scan-json` | yes | Exact `qureddy.scan.v1` file |
| `--pdf` | yes | New PDF output path |
| `--result` | yes | New RenderResult output path |
| `--verbose` | no | Enable informational `log/slog` diagnostics on stderr |
| `--log-format` | no | `text` or `json` diagnostics; default `text` |

## Profile CLI

The forward-compatible profile command makes the report projection explicit:

```bash
breachsafe-pdf version
breachsafe-pdf profile list
breachsafe-pdf profile inspect breachsafe/community
breachsafe-pdf render --profile breachsafe/community \
  --request REQUEST.json \
  --cbom CBOM.json \
  --scan-json SCAN.json \
  --pdf REPORT.pdf \
  --result REPORT.result.json
```

`breachsafe/community` currently selects the
`qureddy-single-scan/v1alpha1` input adapter. Future OSCAL and compliance
profiles must register their own adapter and mapping; the PDF renderer never
parses producer-native bytes.

## Streams and exit codes

| Condition | stdout | stderr | Exit |
| --- | --- | --- | --- |
| `--version` | Version line | Empty | `0` |
| Successful render | One RenderResult JSON document | Empty | `0` |
| Flag or missing path | Empty | Usage or validation error | `2` or typed fault code |
| Admission failure | Empty | Structured fault message | Typed nonzero code |

The CLI writes only the final RenderResult to stdout on success. The PDF is written
to the path supplied by `--pdf`. Diagnostics use Go `log/slog`, default to errors,
and are written only to stderr. Evidence bytes, secrets, and sensitive paths are
not logged.

## Container invocation

The product image has a non-root entrypoint:

```bash
docker run --rm breachsafe-pdf:PINNED --version
```

For rendering, mount a directory at `/work/run` and pass paths inside that mount.
