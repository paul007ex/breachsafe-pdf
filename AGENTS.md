<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# BreachSAFE PDF instructions

This repository owns the reusable Go PDF compiler and its minimal product image. Evidence
collection, OSCAL interpretation, and ePack composition belong to consuming repositories.

Use the pinned shared skills listed by the golden toolchain repository's
[`docs/skills.md`](https://github.com/paul007ex/breachsafe-golden-go/blob/main/docs/skills.md).
The applicable gate sequence is Go engineering, quality review, CI/CD hygiene, container hygiene,
release, and review gate. Add the test-harness skill for golden PDF/render and fuzz fixtures.

The CI image is `ghcr.io/paul007ex/breachsafe-golden-go:1.26.6`; do not copy its Dockerfile into
this repository. Product runtime images remain minimal and non-root.

Run the full local gate sequence with `scripts/run-gates.sh` after the golden image is published.
