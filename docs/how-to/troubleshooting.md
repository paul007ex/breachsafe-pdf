<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Troubleshoot a report run

## Contents

- [`--version` prints the wrong version](#version-prints-the-wrong-version)
- [`schema mismatch`](#schema-mismatch)
- [`digest mismatch`](#digest-mismatch)
- [`correlation mismatch`](#correlation-mismatch)
- [`LIMIT_EXCEEDED`](#limit_exceeded)
- [Docker cannot see `/tmp`](#docker-cannot-see-tmp)
- [PDF renders but is rejected](#pdf-renders-but-is-rejected-by-a-consumer)

## `--version` prints the wrong version

The generator version is injected at build time:

```bash
docker build --build-arg REPORT_VERSION=0.1.1 -t breachsafe-pdf:local .
docker run --rm breachsafe-pdf:local --version
```

If the output is still `0.1.0`, inspect the image tag and rebuild without reusing a
stale image tag.

## `schema mismatch`

Check the request declarations against the producer documents:

```bash
jq -r '.schema_version, .scan.scanner_version' scan.json
jq -r '.bomFormat, .specVersion, .metadata.tools.components[]?.name' scan.cdx.json
```

The current compiler requires `qureddy.scan.v1` and CycloneDX 1.7 JSON.

## `digest mismatch`

Recalculate digests from the exact files passed to the container:

```bash
shasum -a 256 scan.json scan.cdx.json request.json
```

Update the request declarations from those bytes. Do not normalize, pretty-print,
sort, or edit the producer documents after calculating the values.

## `correlation mismatch`

Inspect the producer identities:

```bash
jq '.scan, .target' scan.json
jq '.metadata.properties' scan.cdx.json
```

The common cause is two independent QuReddy invocations. A repeated target name does
not make two scan IDs the same run. Serialize the CBOM from the saved `ScanResult`,
or use the future one-run producer output tracked in QuReddy issue #430.

## `LIMIT_EXCEEDED`

The producer may contain a legitimate long observation list. The compiler keeps
these fields bounded, but a source field can still exceed the configured maximum.
Inspect the field named in the error. Do not truncate producer bytes to make the
report pass; fix the model boundary or produce a bounded source projection with an
explicit limitation.

## Docker cannot see `/tmp`

Some macOS Docker contexts do not share the host `/tmp` directory. Use a path under
the repository or another shared host directory:

```bash
RUN_DIR="$PWD/.run-pecutx"
mkdir -p "$RUN_DIR"
docker run --rm -v "$RUN_DIR:/work/run:rw" breachsafe-pdf:local ...
```

Keep generated runs outside Git tracking. Move them to a versioned external artifact
directory after verification.

## PDF renders but is rejected by a consumer

Inspect the capability results and PDF metadata:

```bash
jq '.capability_results, .warnings' report.result.json
pdfinfo report.pdf
```

The v1 compiler makes no PDF/A, tagging, encryption, signature, or complex-script
shaping claim. A downstream delivery layer must add those capabilities.
