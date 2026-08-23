<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# CLI reference

## Contents

1. [Identity command](#identity-command)
2. [Render command](#render-command)
3. [Streams and exit codes](#streams-and-exit-codes)
4. [Container invocation](#container-invocation)

## Identity command

```bash
breachsafe-report-go --version
```

Output is one line containing the binary name and build version. The version is
embedded at build time and is also written to PDF metadata and RenderResult.

## Render command

```text
breachsafe-report-go
  -request REQUEST.json
  -cbom CBOM.json
  -scan-json SCAN.json
  -pdf REPORT.pdf
  -result REPORT.result.json
```

| Option | Required | Description |
| --- | --- | --- |
| `-version` | no | Print the binary version and exit |
| `-request` | yes | Bounded render request JSON |
| `-cbom` | yes | Exact CycloneDX 1.7 CBOM file |
| `-scan-json` | yes | Exact `qureddy.scan.v1` file |
| `-pdf` | yes | New PDF output path |
| `-result` | yes | New RenderResult output path |

## Streams and exit codes

| Condition | stdout | stderr | Exit |
| --- | --- | --- | --- |
| `--version` | Version line | Empty | `0` |
| Successful render | One RenderResult JSON document | Empty | `0` |
| Flag or missing path | Empty | Usage or validation error | `2` or typed fault code |
| Admission failure | Empty | Structured fault message | Typed nonzero code |

The CLI writes only the final RenderResult to stdout on success. The PDF is written
to the path supplied by `-pdf`.

## Container invocation

The product image has a non-root entrypoint:

```bash
docker run --rm breachsafe-pdf:PINNED --version
```

For rendering, mount a directory at `/work/run` and pass paths inside that mount.
