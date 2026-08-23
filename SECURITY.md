<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# Security policy

[![Security boundary](https://img.shields.io/badge/security-fail--closed-8b0000?style=flat-square)](docs/explanation/security.md)
[![License metadata](https://img.shields.io/badge/license-REUSE-green?style=flat-square)](REUSE.toml)

## Contents

1. [Scope](#scope)
2. [Report a vulnerability](#report-a-vulnerability)
3. [Supported versions](#supported-versions)
4. [Security controls](#security-controls)
5. [Disclosure](#disclosure)

## Scope

This policy covers the Go PDF compiler, its container image, input admission, PDF
rendering, and CI or release workflows. It does not cover QuReddy, OpenSSL, ePack,
OSCAL, or enterprise identity services. Report issues in those components to their
canonical repositories.

## Report a vulnerability

Do not open a public issue for a suspected vulnerability. Use GitHub Security Advisories:

<https://github.com/paul007ex/breachsafe-pdf/security/advisories/new>

Include the affected commit or image digest, impact, reproduction steps, and a safe
contact channel. Remove secrets and private evidence from the report.

## Supported versions

| Version | Support |
| --- | --- |
| `main` | Development support |
| Latest published release | Security fixes targeted |
| Older releases | No backport commitment |

An image tag is not an immutable identity. Include the image digest in a report.

## Security controls

The compiler uses bounded reads, regular-file checks, exact-byte SHA-256 declarations,
fail-closed correlation, embedded assets, a non-root scratch image, no remote resources,
and paired PDF/RenderResult output. CI runs static analysis, dependency verification,
vulnerability checks, and no-skipped-test checks when the configured tools are available.

The v1 PDF profile does not claim signatures, encryption, PDF/A, tagged accessibility,
or endpoint authenticity.

## Disclosure

Maintainers coordinate a fix and disclosure timeline with the reporter. Do not publish
details before the affected release or mitigation is available. Reporter credit is
given unless the reporter requests anonymity.
