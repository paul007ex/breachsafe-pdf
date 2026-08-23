<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->

# How-to guides

These pages solve a named operator task. They assume the reader knows the product
boundary and need a command sequence with expected checks.

## Contents

1. [Container pipeline](container-pipeline.md)
2. [Troubleshooting](troubleshooting.md)

## Reading order

Start with [`../quickstart.md`](../quickstart.md) for the shortest verified path.
Use [container pipeline](container-pipeline.md) for a live producer run and
[troubleshooting](troubleshooting.md) when a gate rejects the inputs or PDF.

## Guide rules

- Commands name their prerequisites and output paths.
- Exact producer bytes are preserved.
- Mismatched CBOM and scan JSON are not silently combined.
- Network observations are supplementary and must retain their run manifest.
