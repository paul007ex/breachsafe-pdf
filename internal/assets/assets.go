// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package assets owns the immutable, embedded BreachSAFE visual assets used by
// the Community report. No report input can select a path or remote resource.
package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
)

type IconName string

const (
	IconShieldCheck IconName = "shield-check"
	IconScan        IconName = "scan"
	IconAlert       IconName = "triangle-alert"
	IconDownload    IconName = "download"
)

var iconSVG = map[IconName][]byte{
	IconShieldCheck: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"><path d="M12 2 C14.5 4 17 5 20 5 L20 13 C20 17 17 20 12 22 C7 20 4 17 4 13 L4 5 C7 5 9.5 4 12 2 Z"/><path d="M8.5 12 L11 14.5 L15.5 9.5"/></svg>`),
	IconScan:        []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"><path d="M3 7 L3 5 Q3 3 5 3 L7 3"/><path d="M17 3 L19 3 Q21 3 21 5 L21 7"/><path d="M21 17 L21 19 Q21 21 19 21 L17 21"/><path d="M7 21 L5 21 Q3 21 3 19 L3 17"/></svg>`),
	IconAlert:       []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"><path d="M12 3 C12.8 3 13.4 3.4 13.9 4.2 L21.5 18 C22.4 19.6 21.3 21 19.5 21 L4.5 21 C2.7 21 1.6 19.6 2.5 18 L10.1 4.2 C10.6 3.4 11.2 3 12 3 Z"/><path d="M12 9 L12 14"/><path d="M12 17 L12.01 17"/></svg>`),
	IconDownload:    []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"><path d="M12 3 L12 15"/><path d="M7 10 L12 15 L17 10"/><path d="M3 15 L3 19 Q3 21 5 21 L19 21 Q21 21 21 19 L21 15"/></svg>`),
}

// IconSVG returns a caller-owned copy of one approved path-only UI icon.
func IconSVG(name IconName) ([]byte, error) {
	data, ok := iconSVG[name]
	if !ok {
		return nil, fmt.Errorf("unknown embedded icon %q", name)
	}
	return slices.Clone(data), nil
}

// HelmetPNG returns caller-owned bytes for the approved 330x330 BreachSAFE
// helmet mark embedded by helmet_generated.go.
func HelmetPNG() []byte {
	return slices.Clone(embeddedHelmetPNG)
}

// BundleDigest identifies the exact PNG and SVG bytes embedded in a build.
func BundleDigest() string {
	hash := sha256.New()
	hash.Write([]byte("helmet.png"))
	hash.Write([]byte{0})
	hash.Write(embeddedHelmetPNG)
	hash.Write([]byte{0})
	for _, name := range []IconName{IconAlert, IconDownload, IconScan, IconShieldCheck} {
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write(iconSVG[name])
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
