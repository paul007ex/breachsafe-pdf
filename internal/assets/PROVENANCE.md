# Embedded visual asset provenance

All assets in this package are BreachSAFE first-party material and retain the
`PolyForm-Noncommercial-1.0.0` project license.

| Embedded asset | Source in the working checkout | Source SHA-256 |
|---|---|---|
| 330x330 helmet PNG | `breachsafe-wizard/src/breachsafe_ux/assets/logo.png` | `398a53c7e8dbc04c66d0e2a881b5bffabae3974fc4d53e10ba43253058d2bf93` |
| shield-check SVG | `breachsafe-wizard/src/breachsafe_ux/assets/icons/shield-check.svg` | Recorded by `assets.BundleDigest` with the other embedded SVG bytes |
| scan SVG | `breachsafe-wizard/src/breachsafe_ux/assets/icons/scan.svg` | Recorded by `assets.BundleDigest` with the other embedded SVG bytes |
| triangle-alert SVG | `breachsafe-wizard/src/breachsafe_ux/assets/icons/triangle-alert.svg` | Recorded by `assets.BundleDigest` with the other embedded SVG bytes |
| download SVG | `breachsafe-wizard/src/breachsafe_ux/assets/icons/download.svg` | Recorded by `assets.BundleDigest` with the other embedded SVG bytes |

The renderer reads only these compiled-in bytes. It does not accept asset paths,
remote URLs, or caller-supplied SVG/PDF drawing programs.

The embedded SVG path geometry is normalized from the first-party icons to the
absolute `M/L/C/Q/Z` subset supported by `fpdf.SVGBasicParse`. This is an asset
adaptation owned here—not a fork, monkey patch, or runtime modification of FPDF.
