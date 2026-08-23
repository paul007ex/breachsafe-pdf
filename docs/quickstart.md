<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Quickstart

This is the shortest complete path from a QuReddy TLS scan to a correlated
BreachSAFE PDF. It uses Docker, writes all files into one run directory, and leaves
the original producer bytes unchanged.

## Prerequisites

- Docker with outbound TCP 443 access.
- `jq` and `shasum` on the host.
- A pinned QuReddy image and a pinned BreachSAFE PDF image.

```bash
QUREDDY_IMAGE="qureddy:PINNED"
PDF_IMAGE="breachsafe-pdf:v0.1.1"
TARGET="pq.cloudflareresearch.com"
RUN_DIR="$PWD/.run-pq-cloudflare"
mkdir -p "$RUN_DIR"
```

## Capture versions

```bash
docker run --rm "$QUREDDY_IMAGE" --version \
  | tee "$RUN_DIR/qureddy.version.txt"
docker run --rm --entrypoint /opt/openssl/bin/openssl \
  "$QUREDDY_IMAGE" version \
  | tee "$RUN_DIR/openssl.version.txt"
docker run --rm "$PDF_IMAGE" --version \
  | tee "$RUN_DIR/pdf.version.txt"
```

These files are run evidence. The PDF compiler does not invoke these commands on
your behalf.

## Run QuReddy once

```bash
docker run --rm \
  -v "$RUN_DIR:/var/lib/qureddy:rw" \
  "$QUREDDY_IMAGE" \
  scan tls "$TARGET" \
  --format json \
  --output /var/lib/qureddy/scan.json \
  --log /var/lib/qureddy/scan.log.jsonl
```

Confirm the scan completed:

```bash
jq -r '[.scan.scan_id, .scan.scanner_version, .summary.readiness,
  .summary.finding_count] | @tsv' "$RUN_DIR/scan.json"
```

## Create the CBOM from that result

Until QuReddy issue [#430](https://github.com/BreachSAFE/qureddy/issues/430) ships,
serialize the CBOM from the saved result with QuReddy's official renderer:

```bash
docker run --rm \
  --entrypoint python \
  -v "$RUN_DIR:/var/lib/qureddy:rw" \
  "$QUREDDY_IMAGE" \
  -c 'import json; from pathlib import Path; from qureddy.core.models import ScanResult; from qureddy.output.cbom import render_cbom; p=Path("/var/lib/qureddy/scan.json"); r=ScanResult.model_validate(json.loads(p.read_text())); f=Path("/var/lib/qureddy/scan.cdx.json").open("w", encoding="utf-8"); render_cbom(r, f); f.close()'
```

## Create the request and render

```bash
CBOM_SHA256="$(shasum -a 256 "$RUN_DIR/scan.cdx.json" | awk '{print $1}')"
SCAN_SHA256="$(shasum -a 256 "$RUN_DIR/scan.json" | awk '{print $1}')"

jq --arg cbom "$CBOM_SHA256" --arg scan "$SCAN_SHA256" \
  '.cbom.expected_sha256=$cbom
   | .scan_json.expected_sha256=$scan
   | .allow_correlation_mismatch=false' \
  examples/community-single-scan.request.json \
  > "$RUN_DIR/request.json"

docker run --rm \
  --user 65532:65532 \
  -v "$RUN_DIR:/work/run:rw" \
  "$PDF_IMAGE" \
  -request /work/run/request.json \
  -cbom /work/run/scan.cdx.json \
  -scan-json /work/run/scan.json \
  -pdf /work/run/report.pdf \
  -result /work/run/report.result.json
```

## Verify and inspect

```bash
jq -r '.input_artifacts[].relationship' "$RUN_DIR/report.result.json"
pdfinfo "$RUN_DIR/report.pdf" | egrep 'Pages|JavaScript|Encrypted'
shasum -a 256 "$RUN_DIR/report.pdf"
```

Expected relationships are both `matched`. Keep the entire run directory together
when handing the evidence to another operator.
