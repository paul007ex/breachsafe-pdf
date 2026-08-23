<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Generate your first report

This tutorial builds the repository image and renders a report from the checked-in
community fixture. It does not contact a network target.

## Contents

1. [Prerequisites](#prerequisites)
2. [Build the image](#build-the-image)
3. [Render the fixture](#render-the-fixture)
4. [Verify the result](#verify-the-result)

## Prerequisites

- Docker with BuildKit enabled.
- A clone of this repository.

## Build the image

```bash
docker build \
  --build-arg REPORT_VERSION=0.1.1 \
  --tag breachsafe-pdf:local \
  .
```

Verify the binary identity:

```bash
docker run --rm breachsafe-pdf:local --version
```

Expected output for this example build:

```text
breachsafe-report-go 0.1.1
```

## Render the fixture

Create a temporary run directory and copy the fixture inputs:

```bash
RUN_DIR="$(mktemp -d)"
cp examples/community-single-scan.request.json "$RUN_DIR/request.json"
cp internal/admission/testdata/success.cbom.json "$RUN_DIR/scan.cdx.json"
cp internal/admission/testdata/success.scan.json "$RUN_DIR/scan.json"
```

Run the compiler. The container runs as UID 65532 and writes only to the mounted
directory:

```bash
docker run --rm \
  --user 65532:65532 \
  -v "$RUN_DIR:/work/run:rw" \
  breachsafe-pdf:local \
  -request /work/run/request.json \
  -cbom /work/run/scan.cdx.json \
  -scan-json /work/run/scan.json \
  -pdf /work/run/report.pdf \
  -result /work/run/report.result.json
```

## Verify the result

```bash
jq -r '[.pdf.sha256, .pdf.bytes, .pdf.pages] | @tsv' \
  "$RUN_DIR/report.result.json"

pdfinfo "$RUN_DIR/report.pdf" | egrep 'Pages|JavaScript|Encrypted'
```

Admission rejects a schema mismatch, an input digest mismatch, an invalid reference,
or a CBOM and scan JSON pair that cannot be correlated. The report remains a
human-readable projection; the input artifacts remain authoritative.
