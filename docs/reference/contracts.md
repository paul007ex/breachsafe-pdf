<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Input and output contracts

## Contents

- [CLI](#cli)
- [Render request](#render-request)
- [Producer documents](#producer-documents)
- [Correlation](#correlation)
- [RenderResult](#renderresult)
- [Limits and unsupported capabilities](#limits-and-unsupported-capabilities)

## CLI

```text
breachsafe-pdf --version
breachsafe-pdf render --profile breachsafe/community \
  --request REQUEST.json \
  --cbom CBOM.json \
  --scan-json SCAN.json \
  --pdf REPORT.pdf \
  --result REPORT.result.json
```

All five render paths are required. Inputs must be regular files. Symlinks and
non-regular paths are rejected.

## Render request

The current request schema is:

```text
breachsafe.report.request.community-single-scan/v1alpha1
```

The request declares:

| Field | Meaning |
| --- | --- |
| `identity` | Report ID, subject name, timestamp, language, classification |
| `cbom` | Media type, CycloneDX schema, expected SHA-256 |
| `scan_json` | Media type, QuReddy schema, expected SHA-256 |
| `render_options` | Page size, timezone, color mode, accessibility profile |
| `allow_correlation_mismatch` | Diagnostic override; false for evidence |

## Producer documents

The scan document must declare `qureddy.scan.v1`. The CBOM must declare CycloneDX
1.7 and the CycloneDX JSON media type. The admission layer checks structure,
enumerations, references, counts, and QuReddy metadata.

## Correlation

Correlation checks compare the producer identity fields represented in the two
documents, including scan ID, target, observation window, and scanner version. A
pair from two independent network scans is not considered correlated even when the
target string is identical.

## RenderResult

The result document records:

- request, model, and render-request digests;
- final PDF path, media type, SHA-256, byte count, and page count;
- generator and FPDF renderer identities;
- font and visual asset bundle digests;
- exact input artifact digests and relationships;
- capability results and limitations.

The final PDF digest is written to the adjacent RenderResult after rendering. It is
not embedded into the PDF itself.

## Limits and unsupported capabilities

The compiler bounds request, scan, CBOM, model, collection, and PDF sizes. The v1
render profile supports A4 or Letter, UTC, color or grayscale, and no accessibility
profile. Tagged PDF, PDF/A, complex-script shaping, encryption, and digital
signatures are reported as unsupported capabilities.
