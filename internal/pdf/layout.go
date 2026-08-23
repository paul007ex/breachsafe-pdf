// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package pdf

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"codeberg.org/go-pdf/fpdf"
	"github.com/paul007ex/breachsafe-pdf/internal/assets"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
)

type color struct{ r, g, b int }

type palette struct {
	navy, navy2, teal, blue, ink, muted, line, paper, white     color
	softBlue, softTeal, softAmber, softRed, softGreen, softGray color
	critical, high, medium, low, info                           color
}

func reportPalette(mode evidence.ColorMode) palette {
	if mode == evidence.ColorGrayscale {
		return palette{
			navy: color{30, 30, 30}, navy2: color{50, 50, 50}, teal: color{76, 76, 76}, blue: color{88, 88, 88},
			ink: color{32, 32, 32}, muted: color{98, 98, 98}, line: color{208, 208, 208}, paper: color{249, 249, 249}, white: color{255, 255, 255},
			softBlue: color{240, 240, 240}, softTeal: color{235, 235, 235}, softAmber: color{232, 232, 232}, softRed: color{225, 225, 225}, softGreen: color{238, 238, 238}, softGray: color{245, 245, 245},
			critical: color{35, 35, 35}, high: color{60, 60, 60}, medium: color{95, 95, 95}, low: color{130, 130, 130}, info: color{165, 165, 165},
		}
	}
	return palette{
		navy: color{12, 27, 48}, navy2: color{18, 48, 76}, teal: color{0, 157, 166}, blue: color{33, 99, 171},
		ink: color{25, 39, 54}, muted: color{83, 99, 116}, line: color{202, 213, 224}, paper: color{248, 250, 252}, white: color{255, 255, 255},
		softBlue: color{235, 244, 255}, softTeal: color{229, 248, 247}, softAmber: color{255, 246, 220}, softRed: color{255, 234, 236}, softGreen: color{231, 248, 238}, softGray: color{242, 245, 248},
		critical: color{146, 20, 45}, high: color{203, 47, 58}, medium: color{225, 139, 22}, low: color{37, 111, 179}, info: color{74, 92, 110},
	}
}

type layout struct {
	ctx         context.Context
	pdf         *fpdf.Fpdf
	model       evidence.CommunitySingleScan
	modelDigest string
	renderer    *Renderer
	icons       map[assets.IconName]fpdf.SVGBasicType
	colors      palette
	pageWidth   float64
	pageHeight  float64
	left        float64
	right       float64
	top         float64
	bottom      float64
	bodyWidth   float64
	section     string
}

func newLayout(ctx context.Context, pdf *fpdf.Fpdf, model evidence.CommunitySingleScan, modelDigest string, renderer *Renderer, icons map[assets.IconName]fpdf.SVGBasicType) *layout {
	width, height := pdf.GetPageSize()
	left, right := 14.0, 14.0
	return &layout{
		ctx: ctx, pdf: pdf, model: model, modelDigest: modelDigest, renderer: renderer, icons: icons,
		colors: reportPalette(model.RenderOptions.ColorMode), pageWidth: width, pageHeight: height,
		left: left, right: right, top: 23, bottom: height - 21, bodyWidth: width - left - right,
	}
}

func (layout *layout) registerFurniture() {
	layout.pdf.SetHeaderFunc(func() {
		layout.fillColor(layout.colors.navy)
		layout.pdf.Rect(0, 0, layout.pageWidth, 17.5, "F")
		layout.drawColor(layout.colors.white)
		layout.pdf.SetLineWidth(0.28)
		layout.drawIcon(assets.IconShieldCheck, layout.left, 4.3, 6.3, "D")
		layout.setFont("B", 9, layout.colors.white)
		layout.pdf.SetXY(layout.left+9, 5.2)
		layout.pdf.CellFormat(70, 4.5, "BREACHSAFE", "", 0, "L", false, 0, "")
		layout.setFont("", 7.2, color{205, 222, 238})
		layout.pdf.SetXY(layout.left+9, 9.3)
		layout.pdf.CellFormat(90, 3.8, "EVIDENCE-BACKED QUANTUM READINESS", "", 0, "L", false, 0, "")
		layout.setFont("B", 7.3, layout.colors.white)
		layout.pdf.SetXY(layout.pageWidth-92, 6.6)
		layout.pdf.CellFormat(78, 4, strings.ToUpper(layout.section), "", 0, "R", false, 0, "")
		layout.pdf.SetY(layout.top)
	})
	layout.pdf.SetFooterFunc(func() {
		y := layout.pageHeight - 15.5
		layout.drawColor(layout.colors.line)
		layout.pdf.SetLineWidth(0.22)
		layout.pdf.Line(layout.left, y, layout.pageWidth-layout.right, y)
		layout.setFont("", 6.4, layout.colors.muted)
		layout.pdf.SetXY(layout.left, y+2)
		layout.pdf.CellFormat(85, 3.6, "Report ID: "+layout.model.Identity.ReportID, "", 0, "L", false, 0, "")
		layout.pdf.SetXY(layout.left+86, y+2)
		layout.pdf.CellFormat(57, 3.6, "Generated "+layout.model.Identity.GeneratedAt.UTC().Format("2006-01-02"), "", 0, "C", false, 0, "")
		layout.pdf.SetXY(layout.pageWidth-layout.right-43, y+2)
		layout.pdf.CellFormat(43, 3.6, fmt.Sprintf("Page %d of {nb}", layout.pdf.PageNo()), "", 0, "R", false, 0, "")
		layout.pdf.SetXY(layout.left, y+6)
		layout.setFont("", 5.9, layout.colors.muted)
		layout.pdf.CellFormat(layout.bodyWidth, 3.2, "Human-readable projection; exact CBOM and scan JSON remain authoritative.", "", 0, "R", false, 0, "")
	})
}

func (layout *layout) compose() error {
	if err := layout.cover(); err != nil {
		return err
	}
	if err := layout.executiveSummary(); err != nil {
		return err
	}
	if err := layout.scopeAndMethod(); err != nil {
		return err
	}
	if err := layout.posture(); err != nil {
		return err
	}
	if err := layout.findings(); err != nil {
		return err
	}
	if err := layout.inventory(); err != nil {
		return err
	}
	if err := layout.gapsAndLimitations(); err != nil {
		return err
	}
	return layout.provenance()
}

func (layout *layout) newPage(section string) error {
	if err := layout.ctx.Err(); err != nil {
		return fault.Wrap(fault.CodeCanceled, "pdf.paginate", err)
	}
	if layout.pdf.PageNo() >= hardMaxPages {
		return fault.Format(fault.CodeLimitExceeded, "pdf.paginate", "pages", "report would exceed hard maximum of %d pages", hardMaxPages)
	}
	layout.section = section
	layout.pdf.AddPage()
	layout.pdf.SetY(layout.top)
	return nil
}

func (layout *layout) ensureSpace(height float64, continuation string) error {
	if layout.pdf.GetY()+height <= layout.bottom {
		return nil
	}
	if err := layout.newPage(continuation); err != nil {
		return err
	}
	return layout.continuationTitle(continuation)
}

func (layout *layout) continuationTitle(title string) error {
	layout.setFont("B", 15, layout.colors.navy)
	layout.pdf.SetXY(layout.left, layout.top+1)
	layout.pdf.CellFormat(layout.bodyWidth, 7, title+" — continued", "", 1, "L", false, 0, "")
	layout.drawColor(layout.colors.teal)
	layout.pdf.SetLineWidth(0.8)
	layout.pdf.Line(layout.left, layout.pdf.GetY()+1, layout.left+30, layout.pdf.GetY()+1)
	layout.pdf.SetY(layout.pdf.GetY() + 5)
	return nil
}

func (layout *layout) pageHeading(kicker, title, subtitle string, icon assets.IconName) {
	y := layout.top + 2
	layout.fillColor(layout.colors.softTeal)
	layout.pdf.RoundedRect(layout.left, y, 10, 10, 2.2, "1234", "F")
	layout.drawColor(layout.colors.teal)
	layout.pdf.SetLineWidth(0.35)
	layout.drawIcon(icon, layout.left+2.2, y+2.1, 5.7, "D")
	layout.setFont("B", 7.3, layout.colors.teal)
	layout.pdf.SetXY(layout.left+14, y)
	layout.pdf.CellFormat(layout.bodyWidth-14, 4, strings.ToUpper(kicker), "", 1, "L", false, 0, "")
	layout.setFont("B", 19, layout.colors.navy)
	layout.pdf.SetX(layout.left + 14)
	layout.pdf.CellFormat(layout.bodyWidth-14, 8, title, "", 1, "L", false, 0, "")
	layout.setFont("", 8.3, layout.colors.muted)
	layout.pdf.SetX(layout.left + 14)
	layout.pdf.MultiCell(layout.bodyWidth-14, 4.2, strings.TrimRight(subtitle, "\r\n"), "", "L", false)
	layout.pdf.SetY(max(layout.pdf.GetY()+5, y+19))
}

func (layout *layout) sectionLabel(label string) {
	layout.setFont("B", 7.2, layout.colors.teal)
	layout.pdf.CellFormat(layout.bodyWidth, 5, strings.ToUpper(label), "", 1, "L", false, 0, "")
	layout.drawColor(layout.colors.line)
	layout.pdf.SetLineWidth(0.22)
	layout.pdf.Line(layout.left, layout.pdf.GetY(), layout.pageWidth-layout.right, layout.pdf.GetY())
	layout.pdf.SetY(layout.pdf.GetY() + 3)
}

func (layout *layout) setFont(style string, size float64, text color) {
	text = layout.outputColor(text)
	layout.pdf.SetFont(fontFamily, style, size)
	layout.pdf.SetTextColor(text.r, text.g, text.b)
}

func (layout *layout) fillColor(value color) {
	value = layout.outputColor(value)
	layout.pdf.SetFillColor(value.r, value.g, value.b)
}

func (layout *layout) drawColor(value color) {
	value = layout.outputColor(value)
	layout.pdf.SetDrawColor(value.r, value.g, value.b)
}

func (layout *layout) outputColor(value color) color {
	if layout.model.RenderOptions.ColorMode != evidence.ColorGrayscale || (value.r == value.g && value.g == value.b) {
		return value
	}
	luminance := (299*value.r + 587*value.g + 114*value.b) / 1000
	return color{luminance, luminance, luminance}
}

func (layout *layout) drawIcon(name assets.IconName, x, y, width float64, style string) {
	icon, ok := layout.icons[name]
	if !ok || icon.Wd == 0 {
		return
	}
	layout.pdf.SetXY(x, y)
	layout.pdf.SVGBasicDraw(&icon, width/icon.Wd, style)
}

func (layout *layout) textHeight(text string, width, lineHeight float64) float64 {
	lines := layout.pdf.SplitText(strings.TrimRight(text, "\r\n"), width)
	if len(lines) == 0 {
		return lineHeight
	}
	return float64(len(lines)) * lineHeight
}

func (layout *layout) fitText(value string, width float64) string {
	value = strings.TrimSpace(value)
	if layout.pdf.GetStringWidth(value) <= width {
		return value
	}
	const suffix = "..."
	runes := []rune(value)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimSpace(string(runes)) + suffix
		if layout.pdf.GetStringWidth(candidate) <= width {
			return candidate
		}
	}
	return ""
}

func (layout *layout) writeText(x, y, width float64, text, style string, size, lineHeight float64, textColor color, align string) float64 {
	layout.setFont(style, size, textColor)
	layout.pdf.SetXY(x, y)
	layout.pdf.MultiCell(width, lineHeight, strings.TrimRight(text, "\r\n"), "", align, false)
	return layout.pdf.GetY()
}

func (layout *layout) card(x, y, width, height float64, fill, border color) {
	layout.fillColor(fill)
	layout.drawColor(border)
	layout.pdf.SetLineWidth(0.24)
	layout.pdf.RoundedRect(x, y, width, height, 2.3, "1234", "DF")
}

func (layout *layout) callout(title, body string, fill, accent color, icon assets.IconName) error {
	layout.setFont("", 8.2, layout.colors.ink)
	bodyHeight := layout.textHeight(body, layout.bodyWidth-20, 4.2)
	height := max(20, bodyHeight+12)
	// Ordinary callouts remain a single card. Source-derived detail that is
	// taller than a fresh content area is fragmented without truncation.
	if height > layout.bottom-layout.top-20 {
		return layout.fragmentedCallout(title, body, fill, accent, icon)
	}
	if err := layout.ensureSpace(height+3, layout.section); err != nil {
		return err
	}
	x, y := layout.left, layout.pdf.GetY()
	layout.card(x, y, layout.bodyWidth, height, fill, fill)
	layout.fillColor(accent)
	layout.pdf.RoundedRect(x, y, 2.5, height, 1.2, "14", "F")
	layout.drawColor(accent)
	layout.drawIcon(icon, x+6, y+6, 6, "D")
	layout.setFont("B", 8, accent)
	layout.pdf.SetXY(x+16, y+4.2)
	layout.pdf.CellFormat(layout.bodyWidth-20, 4, strings.ToUpper(title), "", 1, "L", false, 0, "")
	layout.writeText(x+16, y+9, layout.bodyWidth-20, body, "", 8.2, 4.2, layout.colors.ink, "L")
	layout.pdf.SetY(y + height + 3)
	return nil
}

func (layout *layout) fragmentedCallout(title, body string, fill, accent color, icon assets.IconName) error {
	layout.setFont("", 8.2, layout.colors.ink)
	lines := layout.pdf.SplitText(strings.TrimRight(body, "\r\n"), layout.bodyWidth-20)
	if len(lines) == 0 {
		lines = []string{"Not asserted"}
	}
	part := 1
	for len(lines) > 0 {
		available := layout.bottom - layout.pdf.GetY()
		maximumLines := int((available - 12) / 4.2)
		if maximumLines < 1 {
			if err := layout.newPage(layout.section); err != nil {
				return err
			}
			if err := layout.continuationTitle(layout.section); err != nil {
				return err
			}
			continue
		}
		count := min(maximumLines, len(lines))
		height := max(20, float64(count)*4.2+12)
		x, y := layout.left, layout.pdf.GetY()
		layout.card(x, y, layout.bodyWidth, height, fill, fill)
		layout.fillColor(accent)
		layout.pdf.RoundedRect(x, y, 2.5, height, 1.2, "14", "F")
		layout.drawColor(accent)
		layout.drawIcon(icon, x+6, y+6, 6, "D")
		fragmentTitle := title
		if part > 1 {
			fragmentTitle += fmt.Sprintf(" — continued %d", part)
		}
		layout.setFont("B", 8, accent)
		layout.pdf.SetXY(x+16, y+4.2)
		layout.pdf.CellFormat(layout.bodyWidth-20, 4, strings.ToUpper(fragmentTitle), "", 1, "L", false, 0, "")
		layout.writeText(x+16, y+9, layout.bodyWidth-20, strings.Join(lines[:count], "\n"), "", 8.2, 4.2, layout.colors.ink, "L")
		layout.pdf.SetY(y + height + 3)
		lines = lines[count:]
		part++
		if len(lines) > 0 {
			if err := layout.newPage(layout.section); err != nil {
				return err
			}
			if err := layout.continuationTitle(layout.section); err != nil {
				return err
			}
		}
	}
	return nil
}

func (layout *layout) keyValue(label, value string, width float64) float64 {
	x, y := layout.pdf.GetX(), layout.pdf.GetY()
	labelWidth := min(43.0, width*0.32)
	layout.setFont("B", 7.4, layout.colors.muted)
	layout.pdf.SetXY(x, y)
	layout.pdf.CellFormat(labelWidth, 4.2, strings.ToUpper(label), "", 0, "L", false, 0, "")
	layout.setFont("", 8.2, layout.colors.ink)
	height := layout.textHeight(value, width-labelWidth, 4.2)
	layout.pdf.SetXY(x+labelWidth, y)
	layout.pdf.MultiCell(width-labelWidth, 4.2, strings.TrimRight(value, "\r\n"), "", "L", false)
	layout.pdf.SetY(max(layout.pdf.GetY(), y+height) + 1.2)
	return layout.pdf.GetY()
}

func (layout *layout) severityColor(value string) color {
	switch strings.ToLower(value) {
	case "critical":
		return layout.colors.critical
	case "high":
		return layout.colors.high
	case "medium":
		return layout.colors.medium
	case "low":
		return layout.colors.low
	default:
		return layout.colors.info
	}
}

func (layout *layout) stateFill(value string) color {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "mismatch"), strings.Contains(lower, "failed"), strings.Contains(lower, "vulnerable"), strings.Contains(lower, "critical"), strings.Contains(lower, "high"):
		return layout.colors.softRed
	case strings.Contains(lower, "unknown"), strings.Contains(lower, "partial"), strings.Contains(lower, "unavailable"), strings.Contains(lower, "weak"):
		return layout.colors.softAmber
	case strings.Contains(lower, "hybrid"), strings.Contains(lower, "complete"), strings.Contains(lower, "safe"), strings.Contains(lower, "matched"):
		return layout.colors.softGreen
	default:
		return layout.colors.softBlue
	}
}

func display(value string) string {
	if strings.TrimSpace(value) == "" {
		return "Not asserted"
	}
	return value
}

func truncateRunes(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return string(runes[:maximum-1]) + "…"
}
