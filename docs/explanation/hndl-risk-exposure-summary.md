<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Harvest Now, Decrypt Later - Risk Exposure Summary (proposed)

The current report is a 9-page, evidence-first auditor document. It deliberately refuses a
single verdict. A CISO needs the opposite: one page that answers one question. This proposes an
executive report profile that answers that question and nothing else.

The question: **can traffic recorded from this endpoint today be decrypted once a quantum
computer exists?** That is Harvest Now, Decrypt Later (HNDL). The report communicates the
endpoint's **risk exposure** to it.

## Contents

1. [The one metric](#1-the-one-metric)
2. [Report structure](#2-report-structure)
3. [Where every value comes from](#3-where-every-value-comes-from)
4. [Choosing the finding that matters most](#4-choosing-the-finding-that-matters-most)
5. [Two real examples](#5-two-real-examples)
6. [How it fits the existing code](#6-how-it-fits-the-existing-code)
7. [What is not built yet](#7-what-is-not-built-yet)

## 1. The one metric

The headline is the **risk exposure**, taken directly from QuReddy's `hndl_exposure` axis. No new
computation, no inferred grade. Four levels:

| `hndl_exposure` (tool field) | Risk exposure | Plain meaning |
|---|---|---|
| `at_risk` | **EXPOSED** | Traffic recorded today can be decrypted once a quantum computer exists. |
| `protected_defeasible` | **PARTIALLY EXPOSED** | Hybrid PQ works, but a classical downgrade path remains open. |
| `protected` | **NOT EXPOSED** | No classical fallback observed; recorded traffic is not harvestable. |
| `unknown` | **EXPOSURE UNKNOWN** | Exposure could not be determined from this scan. |

This is the four-level risk scale a CISO reads. It is the tool's own axis relabeled to
risk-exposure language, not a bolt-on score.

## 2. Report structure

```
 HARVEST NOW, DECRYPT LATER - RISK EXPOSURE SUMMARY
 <target>
   RISK EXPOSURE:  <EXPOSED | PARTIALLY EXPOSED | NOT EXPOSED | UNKNOWN>
     <one-line plain meaning>
   Why:            <evaluation.summary + protection>
   What drives it: <the single highest-signal finding, plain>
   Do next:        <recommended_action>
```

One screen. The full findings list, posture axes, and CBOM inventory stay on the detail tabs and
in the 9-page report for the analyst who wants them.

## 3. Where every value comes from

Every printed string is a field QuReddy already emits. Nothing is hand-authored.

| Line | Source field |
|---|---|
| Risk exposure level | `summary.interpretation.hndl_exposure` |
| Plain meaning | fixed label per level (this doc) |
| Why | `interpretation.display.evaluation.summary` + `.protection` |
| What drives it | highest-signal entry of the nuclei JSONL findings |
| Do next | `interpretation.recommended_action` |

## 4. Choosing the finding that matters most

"Highest signal" is computed from finding metadata, never hand-picked:

```
 rank = (severity, readiness weight, confidence)
   severity:   critical > high > medium > low > info
   readiness:  classically_weak > quantum_vulnerable > transitional_hybrid > quantum_safe
   confidence: high > medium > low
```

A VPN scan with an INFO responder line, two LOW classical-KEX lines, and one MEDIUM weak-DH line
surfaces the MEDIUM weak-DH finding. A CISO sees the one that matters, not the tag soup.

## 5. Two real examples

Live `qureddy 0.9.7` scans:

- **mozilla.org** -> `hndl_exposure = at_risk` -> **RISK EXPOSURE: EXPOSED.** Classical X25519 only;
  the hybrid PQ probe did not confirm support. Drives on "Classical X25519 available."
- **pq.cloudflareresearch.com** -> `hndl_exposure = protected_defeasible` -> **PARTIALLY EXPOSED.**
  Hybrid X25519MLKEM768 negotiated, but classical X25519 still accepted.

## 6. How it fits the existing code

No rewrite. The renderer is already pluggable:

- `input.Adapter.Admit()` (the `qureddy` adapter) already builds the model, including
  `PostureAxes`, `Findings`, and the evaluation strings. No change.
- A new `report.Profile` (`executive`) registers alongside `community` and selects a
  three-slot executive view.
- `evidenceapp.RenderFilesProfile` already takes the renderer and profile as parameters, so the
  executive profile pairs the same `qureddy` adapter with an executive renderer that emits one
  page instead of nine.

## 7. What is not built yet

The summary ships today from tool strings alone. Three slots stay empty until their source lands,
and each is shown as pending rather than faked:

1. **Letter grade (A-F)** - needs the QuReddy Score (`score()`), BreachSAFE/qureddy#669.
2. **Per-finding remediation** - needs a remediation catalog keyed by `finding_type`, in
   qureddy-app (interpretation layer).
3. **CVE / CVSS** - needs the sslyze source feeding the nuclei `classification.cve-id` slot.

The risk-exposure headline, the why, the driving finding, and the action are all available now.
