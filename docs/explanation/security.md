<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Security and trust boundaries

## Contents

- [Scope](#scope)
- [Threats addressed](#threats-addressed)
- [Trust statements](#trust-statements)
- [Container boundary](#container-boundary)
- [Correlation override](#correlation-override)
- [Reporting limitations](#reporting-limitations)

## Scope

This page describes the PDF compiler's security boundary. It does not claim that a
QuReddy scan proves endpoint authenticity, that a CBOM is complete, or that a PDF
is a signature or compliance artifact.

## Threats addressed

| Threat | Control |
| --- | --- |
| Oversized producer input | Per-document and total model limits |
| Symlink or special-file input | `Lstat` and regular-file checks |
| Digest substitution | Request-declared SHA-256 comparison |
| CBOM and scan mixing | Producer identity correlation and fail-closed default |
| Remote resource loading | Embedded fonts, icons, and helmet asset only |
| Template or code injection | Typed model and fixed Go layout code |
| Partial output | Paired no-clobber output writer |
| Container privilege escalation | Scratch runtime and UID 65532 |
| PDF active content | Renderer emits no JavaScript, forms, or attachments |

## Trust statements

The compiler can establish:

- which input bytes it admitted;
- which declared schemas and limits passed;
- whether the CBOM and scan JSON correlated under the implemented checks;
- which PDF bytes and RenderResult it produced;
- which embedded asset and font bundles were used.

The compiler cannot establish:

- that a remote endpoint is controlled by the named organization;
- that the producer scan was complete outside its declared coverage;
- that the CBOM is truthful beyond its producer evidence;
- that a PDF recipient is the intended recipient;
- signer identity, authenticity, or non-repudiation.

## Container boundary

The runtime image contains the compiled report binary and embedded assets. It does
not contain QuReddy, OpenSSL, a shell, a network client, or a template interpreter.
The caller mounts an input/output directory and supplies file paths. The container
does not write outside that mount.

## Correlation override

`allow_correlation_mismatch=true` is a diagnostic feature. It produces a visibly
limited projection with mismatch relationships and must not be used for an evidence
package or a compliance assertion.

## Reporting limitations

The v1 profile does not provide tagged PDF accessibility, PDF/A archival conformance,
complex-script shaping, encryption, or digital signatures. These are explicit
capability results in RenderResult, not hidden omissions.
