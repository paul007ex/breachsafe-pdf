// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package pdf

import (
	"fmt"
	"strings"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

// Dark theme, scoped to the executive report only. The community report is unaffected.
var (
	execBG     = color{13, 15, 20}
	execPanel  = color{26, 30, 38}
	execInk    = color{232, 236, 242}
	execMuted  = color{150, 161, 176}
	execTeal   = color{56, 200, 205}
	execWhite  = color{255, 255, 255}
	execHairln = color{48, 54, 66}
	execTotal  = 5
)

// composeExecutive renders the multi-page Harvest Now, Decrypt Later risk exposure
// report for a CISO on dark pages. Every value is a field the producer already emitted;
// the readiness axis it relabels is unchanged and no combined score is computed here.
// Standards references are a deterministic lookup on each finding's rule id.
func (layout *layout) composeExecutive() error {
	if err := layout.execVerdictPage(); err != nil {
		return err
	}
	if err := layout.execFindingsPage(); err != nil {
		return err
	}
	if err := layout.execStandardsPage(); err != nil {
		return err
	}
	if err := layout.execScopePage(); err != nil {
		return err
	}
	return layout.execReferencesPage()
}

// execPageStart begins a dark page with the standard header.
func (layout *layout) execPageStart(section string, page int) error {
	if err := layout.newPage(section); err != nil {
		return err
	}
	layout.fillColor(execBG)
	layout.pdf.Rect(0, 0, layout.pageWidth, layout.pageHeight, "F")
	layout.setFont("B", 8.5, execWhite)
	layout.pdf.SetXY(layout.left, 12)
	layout.pdf.CellFormat(120, 5, "BREACHSAFE", "", 0, "L", false, 0, "")
	layout.setFont("B", 7, execTeal)
	layout.pdf.SetXY(layout.pageWidth-layout.right-100, 12.6)
	layout.pdf.CellFormat(100, 4, fmt.Sprintf("%s · PAGE %d OF %d", strings.ToUpper(section), page, execTotal), "", 0, "R", false, 0, "")
	layout.drawColor(execHairln)
	layout.pdf.SetLineWidth(0.2)
	layout.pdf.Line(layout.left, 19, layout.pageWidth-layout.right, 19)
	return nil
}

// execVerdictPage — page 1: the risk exposure verdict, top items, and why now.
func (layout *layout) execVerdictPage() error {
	if err := layout.execPageStart("Harvest Now Decrypt Later", 1); err != nil {
		return err
	}
	x, w := layout.left, layout.bodyWidth
	label, plain, accent := layout.riskExposure()

	layout.writeText(x, 26, w, layout.model.Subject.DisplayName, "B", 18, 8, execWhite, "L")
	layout.setFont("", 8, execMuted)
	layout.pdf.SetXY(x, 36)
	layout.pdf.CellFormat(w, 5, display(layout.model.Subject.CanonicalAddress)+"   ·   RUN "+layout.model.Run.ID, "", 1, "L", false, 0, "")

	y := 48.0
	layout.fillColor(accent)
	layout.pdf.RoundedRect(x, y, w, 38, 3, "1234", "F")
	layout.setFont("B", 7.5, execWhite)
	layout.pdf.SetXY(x+10, y+7)
	layout.pdf.CellFormat(w-20, 4, "RISK EXPOSURE", "", 1, "L", false, 0, "")
	layout.setFont("B", 30, execWhite)
	layout.pdf.SetXY(x+10, y+12)
	layout.pdf.CellFormat(w-20, 13, label, "", 1, "L", false, 0, "")
	layout.writeText(x+10, y+28, w-20, plain, "", 9, 4.5, execWhite, "L")

	y = 96
	layout.sectionKicker(x, y, "TOP ITEMS FOR ATTENTION")
	y += 7
	top := topFindings(layout.model.Findings.Items, 3)
	if len(top) == 0 {
		layout.writeText(x, y, w, "No findings were produced for this run.", "", 9, 4.5, execMuted, "L")
	}
	for index, finding := range top {
		layout.setFont("B", 10, execInk)
		layout.pdf.SetXY(x, y)
		layout.pdf.CellFormat(w, 6, fmt.Sprintf("%d.  %s", index+1, finding.Title), "", 1, "L", false, 0, "")
		y += 8
	}

	y = 148
	layout.sectionKicker(x, y, "WHY NOW")
	layout.writeText(x, y+7, w,
		"NSA CNSA 2.0 mandates post-quantum migration by 2030. Harvest-now/decrypt-later traffic is decryptable retroactively, so the exposure clock starts the day the traffic is recorded, not the day a quantum computer arrives.",
		"", 9, 4.6, execInk, "L")
	return nil
}

// execFindingsPage — page 2: each finding with what/why/fix and standards references.
func (layout *layout) execFindingsPage() error {
	if err := layout.execPageStart("Findings", 2); err != nil {
		return err
	}
	x, w := layout.left, layout.bodyWidth
	y := 26.0
	findings := topFindings(layout.model.Findings.Items, 4)
	if len(findings) == 0 {
		layout.writeText(x, y, w, "No findings were produced for this run.", "", 9, 4.5, execMuted, "L")
		return nil
	}
	for _, finding := range findings {
		refs, cwe, fix := references.lookup(finding.RuleRef)
		layout.setFont("B", 10, execInk)
		layout.pdf.SetXY(x, y)
		layout.pdf.CellFormat(w-40, 6, "["+strings.ToUpper(display(finding.Severity))+"]  "+finding.Title, "", 0, "L", false, 0, "")
		layout.setFont("", 7, execMuted)
		layout.pdf.SetXY(x+w-40, y+1)
		layout.pdf.CellFormat(40, 5, finding.RuleRef, "", 1, "R", false, 0, "")
		y = layout.writeText(x, y+7, w, finding.Description, "", 8.3, 4, execMuted, "L") + 1
		if fix != "" {
			layout.setFont("B", 8, execInk)
			layout.pdf.SetXY(x, y)
			layout.pdf.CellFormat(14, 4.4, "Fix:", "", 0, "L", false, 0, "")
			y = layout.writeText(x+14, y, w-14, fix, "", 8.2, 4.2, execInk, "L") + 1
		}
		if len(refs) > 0 || cwe != "" {
			layout.setFont("B", 7.4, execTeal)
			layout.pdf.SetXY(x, y)
			line := "Refs: " + strings.Join(refs, " · ")
			if cwe != "" {
				line += "        " + cwe
			}
			layout.pdf.CellFormat(w, 4, line, "", 1, "L", false, 0, "")
			y += 5
		}
		layout.drawColor(execHairln)
		layout.pdf.SetLineWidth(0.2)
		layout.pdf.Line(x, y, x+w, y)
		y += 5
	}
	return nil
}

// execReferencesPage — page 5: the external standards and resources cited in the report.
func (layout *layout) execReferencesPage() error {
	if err := layout.execPageStart("References and standards", 5); err != nil {
		return err
	}
	x, w := layout.left, layout.bodyWidth
	y := 26.0
	layout.writeText(x, y, w, "External standards and resources cited in this report. These are published references, not endorsements.", "", 8, 4.2, execMuted, "L")
	y += 10
	for _, resource := range references.Resources {
		layout.setFont("B", 8.4, execInk)
		layout.pdf.SetXY(x, y)
		layout.pdf.CellFormat(w, 4.4, resource.Name, "", 1, "L", false, 0, "")
		layout.setFont("", 7.2, execTeal)
		layout.pdf.SetXY(x, y+4.2)
		layout.pdf.CellFormat(w, 3.6, resource.URL, "", 1, "L", false, 0, "")
		y += 10
	}
	return nil
}

// execStandardsPage — page 3: alignment against the migration standards, from the axes.
func (layout *layout) execStandardsPage() error {
	if err := layout.execPageStart("Standards alignment", 3); err != nil {
		return err
	}
	x, w := layout.left, layout.bodyWidth
	readiness := layout.readinessValue()
	pqKex := readiness == "transitional_hybrid" || readiness == "quantum_safe"
	unknownKex := readiness == "" || readiness == "unknown"
	hygieneOK := layout.hygieneAcceptable()

	kexState := "GAP"
	if pqKex {
		kexState = "MET"
	} else if unknownKex {
		kexState = "UNKNOWN"
	}
	hygieneState := "GAP"
	if hygieneOK {
		hygieneState = "MET"
	}

	rows := []struct{ std, req, state string }{
		{"CNSA 2.0 (NSA)", "Post-quantum key exchange by 2030", kexState},
		{"FIPS 203 (ML-KEM)", "Hybrid or PQ key exchange on the wire", kexState},
		{"FIPS 204 (ML-DSA)", "Post-quantum certificate signatures", "GAP"},
		{"NIST SP 800-52 Rev 2", "TLS 1.2+ only, no TLS 1.0/1.1", hygieneState},
		{"RFC 9395 / SP 800-77", "IKEv2 only, no IKEv1 (IPsec)", "N/A"},
	}
	y := 28.0
	layout.setFont("B", 7, execMuted)
	layout.pdf.SetXY(x, y)
	layout.pdf.CellFormat(w*0.30, 5, "STANDARD", "", 0, "L", false, 0, "")
	layout.pdf.CellFormat(w*0.52, 5, "REQUIREMENT", "", 0, "L", false, 0, "")
	layout.pdf.CellFormat(w*0.18, 5, "THIS ENDPOINT", "", 1, "R", false, 0, "")
	y += 7
	for _, row := range rows {
		layout.drawColor(execHairln)
		layout.pdf.SetLineWidth(0.15)
		layout.pdf.Line(x, y-1.5, x+w, y-1.5)
		layout.setFont("B", 8.5, execInk)
		layout.pdf.SetXY(x, y)
		layout.pdf.CellFormat(w*0.30, 6, row.std, "", 0, "L", false, 0, "")
		layout.setFont("", 8.2, execMuted)
		layout.pdf.CellFormat(w*0.52, 6, row.req, "", 0, "L", false, 0, "")
		layout.setFont("B", 8.5, layout.stateColor(row.state))
		layout.pdf.CellFormat(w*0.18, 6, row.state, "", 1, "R", false, 0, "")
		y += 9
	}
	layout.setFont("", 7.6, execMuted)
	layout.pdf.SetXY(x, y+4)
	layout.pdf.CellFormat(w, 4, "NIST deprecates 112-bit classical strength by 2030 and disallows it by 2035.", "", 1, "L", false, 0, "")
	return nil
}

// execScopePage — page 4: what was assessed, coverage, evidence, tools, boundary.
func (layout *layout) execScopePage() error {
	if err := layout.execPageStart("Scope and method", 4); err != nil {
		return err
	}
	x, w := layout.left, layout.bodyWidth
	y := 28.0
	tool := "QuReddy"
	if len(layout.model.Tools) > 0 {
		tool = display(layout.model.Tools[0].Name) + " " + display(layout.model.Tools[0].Version)
	}
	qv := 0
	for _, asset := range layout.model.Inventory.Items {
		if strings.EqualFold(asset.Readiness, "quantum_vulnerable") {
			qv++
		}
	}
	rows := [][2]string{
		{"Assessed", fmt.Sprintf("%s (%s)", layout.model.Subject.CanonicalAddress, display(layout.model.Subject.Kind))},
		{"Coverage", fmt.Sprintf("%d attempted / %d completed", layout.model.Coverage.Attempted, layout.model.Coverage.Completed)},
		{"Inventory", fmt.Sprintf("%d crypto assets, %d quantum-vulnerable (CBOM)", layout.model.Inventory.TotalCount, qv)},
		{"Correlation", "CBOM <-> scan " + display(layout.model.Correlation.State)},
		{"Tool", tool},
	}
	for _, row := range rows {
		layout.setFont("B", 8, execMuted)
		layout.pdf.SetXY(x, y)
		layout.pdf.CellFormat(35, 6, row[0], "", 0, "L", false, 0, "")
		layout.setFont("", 8.5, execInk)
		layout.pdf.CellFormat(w-35, 6, row[1], "", 1, "L", false, 0, "")
		y += 8
	}
	layout.drawColor(execHairln)
	layout.pdf.SetLineWidth(0.2)
	layout.pdf.Line(x, y+2, x+w, y+2)
	layout.setFont("B", 7, execMuted)
	layout.pdf.SetXY(x, y+5)
	layout.pdf.CellFormat(w, 4, "DECISION-USE BOUNDARY", "", 1, "L", false, 0, "")
	layout.writeText(x, y+10, w, "Reports the harvest-now/decrypt-later risk exposure observed on this endpoint in this run. Not a certification; not organization-wide. Exact CBOM and scan JSON remain authoritative.", "", 7.8, 4, execMuted, "L")
	return nil
}

// sectionKicker draws a small teal uppercase label.
func (layout *layout) sectionKicker(x, y float64, label string) {
	layout.setFont("B", 7.2, execTeal)
	layout.pdf.SetXY(x, y)
	layout.pdf.CellFormat(200, 4, label, "", 1, "L", false, 0, "")
}

// stateColor maps a MET/GAP/UNKNOWN/N/A alignment state to a color.
func (layout *layout) stateColor(state string) color {
	switch state {
	case "MET":
		return execTeal
	case "GAP":
		return layout.severityColor("high")
	default:
		return execMuted
	}
}

// hygieneAcceptable reports whether the protocol-hygiene axis is acceptable.
func (layout *layout) hygieneAcceptable() bool {
	for _, axis := range layout.model.PostureAxes {
		if strings.Contains(strings.ToLower(axis.Label), "hygiene") {
			v := strings.ToLower(axis.Value)
			return strings.Contains(v, "acceptable") || strings.Contains(v, "ok")
		}
	}
	return false
}

// readinessValue returns the producer readiness value from the readiness posture axis.
func (layout *layout) readinessValue() string {
	for _, axis := range layout.model.PostureAxes {
		lower := strings.ToLower(axis.Label)
		if strings.Contains(lower, "readiness") || strings.Contains(lower, "quantum") {
			return strings.ToLower(axis.Value)
		}
	}
	return ""
}

// riskExposure maps the producer readiness value to a plain risk-exposure verdict.
func (layout *layout) riskExposure() (label, plain string, accent color) {
	switch layout.readinessValue() {
	case "quantum_vulnerable", "classically_weak":
		return "EXPOSED",
			"Traffic recorded today can be decrypted once a quantum computer exists.",
			layout.severityColor("high")
	case "transitional_hybrid":
		return "PARTIALLY EXPOSED",
			"Hybrid post-quantum protection is present, but a classical downgrade path remains.",
			layout.severityColor("medium")
	case "quantum_safe":
		return "NOT EXPOSED",
			"No classical fallback was observed; recorded traffic is not harvestable.",
			layout.severityColor("info")
	default:
		return "EXPOSURE UNKNOWN",
			"Harvest-now/decrypt-later exposure could not be determined from this run.",
			color{90, 96, 108}
	}
}

// topFindings returns up to limit findings, highest severity first.
func topFindings(findings []evidence.Finding, limit int) []evidence.Finding {
	rank := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1, "info": 0}
	sorted := make([]evidence.Finding, len(findings))
	copy(sorted, findings)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && rank[strings.ToLower(sorted[j].Severity)] > rank[strings.ToLower(sorted[j-1].Severity)]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}
