<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# RenderResult reference

RenderResult is the machine-readable receipt adjacent to every successful PDF.
It is not a replacement for the input CBOM or scan JSON.

## Top-level fields

| Field | Meaning |
| --- | --- |
| `schema_version` | RenderResult contract identifier |
| `report_id` | Request report identity |
| `contract_version` | Internal report-model contract |
| `input_contract` | Accepted producer-input contract |
| `view` | Rendered view name |
| `generated_at` | Request generation time |
| `admission_request_sha256` | Exact request bytes admitted |
| `model_sha256` | Canonical source-neutral model digest |
| `render_request_sha256` | Canonical render-model digest |
| `pdf` | Final PDF path, digest, size, and page count |
| `generator` | PDF compiler name, version, and optional commit |
| `renderer` | FPDF implementation and version |
| `font_bundle` | Embedded font bundle identity and digest |
| `asset_bundle` | Embedded visual bundle identity and digest |
| `input_artifacts` | CBOM and scan declarations, digests, and relationships |
| `capability_results` | Supported, unsupported, or failed PDF capabilities |
| `warnings` | Human-readable limitations derived from capabilities |

## Input artifact relationship

Each input artifact records a relationship such as:

```text
correlated with scan-json: matched
correlated with cbom: matched
```

Evidence packaging should require `matched` for every relationship. A mismatched
relationship is a diagnostic result and must be treated as a failed packaging gate.

## Capability result statuses

| Status | Meaning |
| --- | --- |
| `pass` | Capability was present and verified during rendering |
| `not_supported` | The v1 renderer makes no claim for that capability |
| `fail` | A capability was expected but rendering did not satisfy it |

## Digest interpretation

SHA-256 values identify exact bytes supplied to or emitted by the compiler. They do
not prove signer identity, endpoint ownership, producer honesty, evidence
completeness, or freshness.
