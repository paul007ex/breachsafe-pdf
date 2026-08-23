<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Run a QuReddy to PDF container pipeline

This procedure produces a report for a TLS target. It requires network access to the
target and uses two containers: the QuReddy collector and the BreachSAFE PDF compiler.

## Contents

- [Capture tool versions](#1-capture-tool-versions)
- [Run one QuReddy scan](#2-run-one-qureddy-scan)
- [Serialize the CBOM](#3-serialize-the-cbom-from-the-same-result)
- [Build the render request](#4-build-the-render-request)
- [Render and verify](#5-render-and-verify)

## 1. Capture tool versions

Capture versions before the scan. Store the command output with the run:

```bash
docker run --rm qureddy:PINNED --version
docker run --rm --entrypoint /opt/openssl/bin/openssl \
  qureddy:PINNED version
docker run --rm breachsafe-pdf:PINNED --version
```

Use an immutable image digest for production runs. Tags are convenient for local
development but do not identify the image contents.

## 2. Run one QuReddy scan

```bash
docker run --rm \
  -v "$RUN_DIR:/var/lib/qureddy:rw" \
  qureddy:PINNED \
  scan tls TARGET \
  --format json \
  --output /var/lib/qureddy/scan.json \
  --log /var/lib/qureddy/scan.log.jsonl
```

The JSON contains the scan ID, target, observations, findings, summary, local
collector dependencies, and QuReddy scanner version.

## 3. Serialize the CBOM from the same result

The CBOM must be serialized from the same in-memory scan result. Running a second
network scan with `--format cbom` creates a different scan ID and is not a correlated
evidence pair. QuReddy issue [#430](https://github.com/BreachSAFE/qureddy/issues/430)
tracks a producer-native one-run command that emits both documents.

Until that command exists, use the official QuReddy renderer on the saved result:

```bash
docker run --rm \
  --entrypoint python \
  -v "$RUN_DIR:/var/lib/qureddy:rw" \
  qureddy:PINNED \
  -c 'import json; from pathlib import Path; from qureddy.core.models import ScanResult; from qureddy.output.cbom import render_cbom; p=Path("/var/lib/qureddy/scan.json"); r=ScanResult.model_validate(json.loads(p.read_text())); f=Path("/var/lib/qureddy/scan.cdx.json").open("w", encoding="utf-8"); render_cbom(r, f); f.close()'
```

This is a temporary orchestration step. The generated CBOM retains the scan ID and
producer timestamps from `scan.json`.

## 4. Build the render request

The request declares the exact SHA-256 values that admission must accept:

```bash
CBOM_SHA256="$(shasum -a 256 "$RUN_DIR/scan.cdx.json" | awk '{print $1}')"
SCAN_SHA256="$(shasum -a 256 "$RUN_DIR/scan.json" | awk '{print $1}')"

jq --arg cbom "$CBOM_SHA256" --arg scan "$SCAN_SHA256" \
  '.cbom.expected_sha256=$cbom
   | .scan_json.expected_sha256=$scan
   | .allow_correlation_mismatch=false' \
  examples/community-single-scan.request.json \
  > "$RUN_DIR/request.json"
```

In a production orchestrator, identity, tool versions, image digests, commands, and
input digests should be recorded in a versioned run manifest alongside this request.

## 5. Render and verify

```bash
docker run --rm \
  --user 65532:65532 \
  -v "$RUN_DIR:/work/run:rw" \
  breachsafe-pdf:PINNED \
  -request /work/run/request.json \
  -cbom /work/run/scan.cdx.json \
  -scan-json /work/run/scan.json \
  -pdf /work/run/report.pdf \
  -result /work/run/report.result.json
```

Verify both relationships are `matched`:

```bash
jq -r '.input_artifacts[].relationship' "$RUN_DIR/report.result.json"
```

Do not set `allow_correlation_mismatch=true` for an evidence package. That mode is a
diagnostic projection and must be labelled as such.
