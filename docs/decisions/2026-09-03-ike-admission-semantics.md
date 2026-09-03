<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Admit IKE evidence without falsifying scan outcomes

## Contents

1. [Context](#context)
2. [Options](#options)
3. [Steelman](#steelman)
4. [Decision](#decision)
5. [Pressure-test evidence](#pressure-test-evidence)

## Context

The PDF admission boundary accepted only TLS and SSH evidence, so a valid QuReddy IKE scan could
not produce either the technical or executive report. IKE also has terminal outcomes such as
`rejected` and `no_response` that describe a completed observation; treating either as a scanner
crash would misstate the evidence.

## Options

Weights: evidence fidelity 40, bounded change 25, future extensibility 20, testability 15.

| Option | Fidelity | Bounded change | Extensibility | Testability | Weighted score |
| --- | ---: | ---: | ---: | ---: | ---: |
| A. Add explicit IKE admission and outcome semantics | 10 | 10 | 8 | 10 | 96 |
| B. Relabel IKE evidence as TLS before admission | 2 | 8 | 2 | 8 | 44 |
| C. Replace admission with a generic scanner registry now | 10 | 3 | 10 | 7 | 76 |

## Steelman

Option B is the smallest mechanical change and reuses a proven report path. It is rejected because
the report would claim the wrong protocol and erase the distinction between transport security and
VPN key exchange.

Option C gives later scanners a clean extension point. It is rejected for this patch because the
current domain is three known scanners, and a registry would expand the trusted admission boundary
without another concrete protocol to validate its abstraction.

## Decision

Choose Option A. Admit `ike` explicitly, map its terminal outcomes at the admission boundary, and
keep the report model protocol-neutral. `completed` and `rejected` are completed observations;
`no_response` is a partial observation, not a tool failure. Preserve fail-closed validation for
unknown scanner names and statuses.

## Pressure-test evidence

The pre-change binary rejected the real IKE fixture first on scanner name and then on status. The
patched binary admits the same QuReddy scan and CBOM, renders a nine-page report, and records the run
as completed with complete coverage. Unit tests pin successful, rejected, and no-response behavior
using the real scanner artifacts.
