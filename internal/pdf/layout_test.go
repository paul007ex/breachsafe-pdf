// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package pdf

import (
	"strings"
	"testing"

	"codeberg.org/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
)

func TestFitTextRespectsMeasuredCellWidth(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes(fontFamily, "B", gobold.TTF)
	pdf.SetFont(fontFamily, "B", 7.8)
	layout := &layout{pdf: pdf}

	const width = 40.0
	value := "X25519MLKEM768, X25519, ECDHE-ECDSA-AES256-GCM-SHA384"
	got := layout.fitText(value, width)
	if pdf.GetStringWidth(got) > width {
		t.Fatalf("fitted width = %.2f, maximum = %.2f: %q", pdf.GetStringWidth(got), width, got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("fitted text = %q, want an explicit truncation suffix", got)
	}
	if unchanged := layout.fitText("X25519", width); unchanged != "X25519" {
		t.Fatalf("short value changed to %q", unchanged)
	}
}
