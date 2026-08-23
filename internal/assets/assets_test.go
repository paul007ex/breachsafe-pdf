// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package assets

import (
	"bytes"
	"image/png"
	"testing"

	"codeberg.org/go-pdf/fpdf"
)

func TestEmbeddedAssetsDecodeAndParse(t *testing.T) {
	configuration, err := png.DecodeConfig(bytes.NewReader(HelmetPNG()))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Width != 330 || configuration.Height != 330 {
		t.Fatalf("favicon dimensions = %dx%d", configuration.Width, configuration.Height)
	}
	for _, name := range []IconName{IconShieldCheck, IconScan, IconAlert, IconDownload} {
		data, err := IconSVG(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fpdf.SVGBasicParse(data); err != nil {
			t.Fatalf("SVGBasicParse(%s): %v", name, err)
		}
	}
	digest := BundleDigest()
	if len(digest) != 64 {
		t.Fatalf("bundle digest = %q", digest)
	}
}

func TestReturnedAssetsAreCallerOwned(t *testing.T) {
	first, err := IconSVG(IconShieldCheck)
	if err != nil {
		t.Fatal(err)
	}
	first[0] = 0
	second, err := IconSVG(IconShieldCheck)
	if err != nil {
		t.Fatal(err)
	}
	if second[0] == 0 {
		t.Fatal("IconSVG returned shared mutable bytes")
	}
	firstPNG := HelmetPNG()
	firstPNG[0] = 0
	if HelmetPNG()[0] == 0 {
		t.Fatal("HelmetPNG returned shared mutable bytes")
	}
}
