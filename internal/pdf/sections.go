// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package pdf

import (
	"fmt"
	"strings"

	"codeberg.org/go-pdf/fpdf"
	"github.com/paul007ex/breachsafe-pdf/internal/assets"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

func (layout *layout) cover() error {
	if err := layout.newPage("Report identity"); err != nil {
		return err
	}
	y := layout.top + 3
	layout.fillColor(layout.colors.navy2)
	layout.pdf.RoundedRect(layout.left, y, layout.bodyWidth, 96, 4, "1234", "F")
	layout.fillColor(layout.colors.teal)
	layout.pdf.RoundedRect(layout.left, y, 4, 96, 2, "14", "F")
	layout.pdf.ImageOptions(helmetImageName, layout.pageWidth-layout.right-52, y+12, 36, 36, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	layout.setFont("B", 7.5, color{114, 224, 218})
	layout.pdf.SetXY(layout.left+12, y+11)
	layout.pdf.CellFormat(120, 5, "COMMUNITY SINGLE-SCAN / EVIDENCE REPORT", "", 1, "L", false, 0, "")
	layout.writeText(layout.left+12, y+21, layout.bodyWidth-65, layout.model.Identity.Name, "B", 22, 8.5, layout.colors.white, "L")
	layout.setFont("B", 7, color{114, 224, 218})
	layout.pdf.SetXY(layout.left+12, y+56)
	layout.pdf.CellFormat(layout.bodyWidth-24, 4, "PREPARED FOR  "+strings.ToUpper(display(layout.model.Identity.OrganizationDisplayName)), "", 1, "L", false, 0, "")
	layout.setFont("", 8, color{201, 220, 235})
	layout.pdf.SetXY(layout.left+12, y+64)
	layout.pdf.CellFormat(50, 4, "PRIMARY SUBJECT", "", 0, "L", false, 0, "")
	layout.setFont("B", 11, layout.colors.white)
	layout.pdf.SetXY(layout.left+12, y+70)
	layout.pdf.CellFormat(layout.bodyWidth-24, 6, layout.model.Subject.DisplayName, "", 1, "L", false, 0, "")
	layout.setFont("", 8.2, color{201, 220, 235})
	layout.pdf.SetX(layout.left + 12)
	layout.pdf.CellFormat(layout.bodyWidth-24, 5, layout.model.Subject.CanonicalAddress, "", 1, "L", false, 0, "")
	layout.setFont("", 6.5, color{201, 220, 235})
	layout.pdf.SetX(layout.left + 12)
	layout.pdf.CellFormat(layout.bodyWidth-24, 4, "RUN "+layout.model.Run.ID+"  ·  REPORT "+layout.model.Identity.ReportID, "", 1, "L", false, 0, "")

	y = layout.top + 108
	layout.setFont("B", 7.2, layout.colors.teal)
	layout.pdf.SetXY(layout.left, y)
	layout.pdf.CellFormat(layout.bodyWidth, 5, "EXECUTIVE SNAPSHOT", "", 1, "L", false, 0, "")
	y += 8
	cardGap := 4.0
	cardWidth := (layout.bodyWidth - 2*cardGap) / 3
	metrics := []struct{ label, value, detail string }{
		{"Run status", strings.ToUpper(string(layout.model.Run.Status)), "Producer state preserved"},
		{"Coverage", strings.ToUpper(string(layout.model.Coverage.Status)), fmt.Sprintf("%d attempted / %d completed", layout.model.Coverage.Attempted, layout.model.Coverage.Completed)},
		{"Highest severity", strings.ToUpper(highestSeverity(layout.model.Findings.Items)), fmt.Sprintf("%d producer findings", layout.model.Findings.TotalCount)},
	}
	for index, metric := range metrics {
		x := layout.left + float64(index)*(cardWidth+cardGap)
		layout.card(x, y, cardWidth, 29, layout.stateFill(metric.value), layout.colors.line)
		layout.setFont("B", 6.8, layout.colors.muted)
		layout.pdf.SetXY(x+4, y+4)
		layout.pdf.CellFormat(cardWidth-8, 4, strings.ToUpper(metric.label), "", 1, "L", false, 0, "")
		layout.setFont("B", 11.5, layout.colors.navy)
		layout.pdf.SetX(x + 4)
		layout.pdf.CellFormat(cardWidth-8, 7, metric.value, "", 1, "L", false, 0, "")
		layout.setFont("", 6.8, layout.colors.muted)
		layout.pdf.SetX(x + 4)
		layout.pdf.CellFormat(cardWidth-8, 4, metric.detail, "", 1, "L", false, 0, "")
	}

	y += 37
	layout.setFont("B", 7.2, layout.colors.teal)
	layout.pdf.SetXY(layout.left, y)
	layout.pdf.CellFormat(layout.bodyWidth, 5, "SELECTED READINESS AXES", "", 1, "L", false, 0, "")
	y += 8
	for index, axis := range selectedCoverAxes(layout.model.PostureAxes) {
		rowY := y + float64(index)*14
		valueX := layout.left + 65
		authorityX := layout.pageWidth - layout.right - 28
		valueWidth := authorityX - valueX - 3
		layout.fillColor(layout.colors.paper)
		layout.pdf.RoundedRect(layout.left, rowY, layout.bodyWidth, 11, 2, "1234", "F")
		layout.setFont("B", 7.8, layout.colors.ink)
		layout.pdf.SetXY(layout.left+4, rowY+2.2)
		layout.pdf.CellFormat(60, 5, axis.Label, "", 0, "L", false, 0, "")
		layout.setFont("B", 7.8, layout.colors.blue)
		layout.pdf.SetXY(valueX, rowY+2.2)
		layout.pdf.CellFormat(valueWidth, 5, layout.fitText(axis.Value, valueWidth), "", 0, "L", false, 0, "")
		layout.setFont("", 6.5, layout.colors.muted)
		layout.pdf.SetXY(authorityX, rowY+2.2)
		layout.pdf.CellFormat(24, 5, string(axis.Authority), "", 0, "R", false, 0, "")
	}

	noticeY := min(layout.bottom-23, y+47)
	layout.card(layout.left, noticeY, layout.bodyWidth, 20, layout.colors.softAmber, layout.colors.softAmber)
	layout.drawColor(layout.colors.medium)
	layout.drawIcon(assets.IconAlert, layout.left+5, noticeY+6, 6, "D")
	layout.setFont("B", 7.3, layout.colors.medium)
	layout.pdf.SetXY(layout.left+15, noticeY+3.2)
	layout.pdf.CellFormat(layout.bodyWidth-19, 4, "EVIDENCE AUTHORITY", "", 1, "L", false, 0, "")
	layout.writeText(layout.left+15, noticeY+8, layout.bodyWidth-20, "This is an evidence-backed assessment, not a certification. Exact producer artifacts remain authoritative; the final PDF hash is external.", "", 7.5, 3.8, layout.colors.ink, "L")
	return nil
}

func (layout *layout) executiveSummary() error {
	if err := layout.newPage("Executive summary"); err != nil {
		return err
	}
	layout.pageHeading("One run / one subject", "Executive summary", "What the admitted evidence says, how much was assessed, and where decision use must stop.", assets.IconShieldCheck)

	y := layout.pdf.GetY()
	gap := 4.0
	width := (layout.bodyWidth - gap) / 2
	metrics := []struct{ label, value, detail string }{
		{"Producer readiness", strings.ToUpper(summaryReadiness(layout.model)), "Tool-asserted; no combined score inferred"},
		{"Correlation", strings.ToUpper(layout.model.Correlation.State), "CBOM ↔ scan JSON"},
		{"Finding inventory", fmt.Sprintf("%d / %d", layout.model.Findings.TotalCount, layout.model.Inventory.TotalCount), "Findings / CBOM crypto assets"},
		{"Observation window", durationDisplay(layout.model.Run.DurationMilliseconds), formatTime(layout.model.Run.StartedAt)},
	}
	for index, metric := range metrics {
		x := layout.left + float64(index%2)*(width+gap)
		rowY := y + float64(index/2)*31
		layout.card(x, rowY, width, 27, layout.stateFill(metric.value), layout.colors.line)
		layout.setFont("B", 6.8, layout.colors.muted)
		layout.pdf.SetXY(x+4, rowY+3.2)
		layout.pdf.CellFormat(width-8, 4, strings.ToUpper(metric.label), "", 1, "L", false, 0, "")
		layout.setFont("B", 12.5, layout.colors.navy)
		layout.pdf.SetX(x + 4)
		layout.pdf.CellFormat(width-8, 7, truncateRunes(metric.value, 48), "", 1, "L", false, 0, "")
		layout.setFont("", 6.7, layout.colors.muted)
		layout.pdf.SetX(x + 4)
		layout.pdf.CellFormat(width-8, 4, truncateRunes(metric.detail, 76), "", 1, "L", false, 0, "")
	}
	layout.pdf.SetY(y + 67)

	layout.sectionLabel("Finding distribution")
	counts := severityCounts(layout.model.Findings.Items)
	total := max(1, layout.model.Findings.TotalCount)
	barX, barY, barWidth := layout.left, layout.pdf.GetY()+1, layout.bodyWidth
	currentX := barX
	for _, severity := range []string{"critical", "high", "medium", "low", "info"} {
		count := counts[severity]
		if count == 0 {
			continue
		}
		segment := barWidth * float64(count) / float64(total)
		layout.fillColor(layout.severityColor(severity))
		layout.pdf.Rect(currentX, barY, segment, 5, "F")
		currentX += segment
	}
	if layout.model.Findings.TotalCount == 0 {
		layout.fillColor(layout.colors.softGray)
		layout.pdf.Rect(barX, barY, barWidth, 5, "F")
	}
	layout.pdf.SetY(barY + 8)
	for _, severity := range []string{"critical", "high", "medium", "low", "info"} {
		layout.setFont("B", 7, layout.severityColor(severity))
		layout.pdf.CellFormat(layout.bodyWidth/5, 4, fmt.Sprintf("%s %d", strings.ToUpper(severity), counts[severity]), "", 0, "L", false, 0, "")
	}
	layout.pdf.Ln(9)

	layout.sectionLabel("Coverage")
	coverage := layout.model.Coverage
	layout.setFont("", 8.2, layout.colors.ink)
	layout.keyValue("Status", strings.ToUpper(string(coverage.Status)), layout.bodyWidth)
	layout.keyValue("Counts", fmt.Sprintf("%d requested · %d attempted · %d completed", coverage.Requested, coverage.Attempted, coverage.Completed), layout.bodyWidth)
	layout.keyValue("Derivation", coverage.Derivation, layout.bodyWidth)
	if len(coverage.NotAssessed) > 0 {
		layout.keyValue("Not assessed", strings.Join(coverage.NotAssessed, "; "), layout.bodyWidth)
	}
	layout.pdf.SetY(layout.pdf.GetY() + 3)
	if err := layout.callout("Correlation: "+layout.model.Correlation.State, layout.model.Correlation.Basis, layout.stateFill(layout.model.Correlation.State), layout.severityColor(correlationSeverity(layout.model.Correlation.State)), assets.IconScan); err != nil {
		return err
	}
	return layout.callout("Decision-use boundary", "Use this report to understand this admitted run and its evidence quality. Do not use it as proof of organization-wide coverage, compliance, authenticity, freshness, or future security.", layout.colors.softBlue, layout.colors.blue, assets.IconAlert)
}

func (layout *layout) scopeAndMethod() error {
	if err := layout.newPage("Scope and method"); err != nil {
		return err
	}
	layout.pageHeading("Collection context", "Scope, method and tools", "The declared target, observation window, producer identities, collector capability, and admitted input schemas.", assets.IconScan)

	layout.sectionLabel("Primary subject and run")
	if err := layout.ensureSpace(48, "Scope and method"); err != nil {
		return err
	}
	x, y := layout.left, layout.pdf.GetY()
	layout.card(x, y, layout.bodyWidth, 44, layout.colors.paper, layout.colors.line)
	layout.pdf.SetXY(x+5, y+5)
	layout.keyValue("Subject", layout.model.Subject.DisplayName, layout.bodyWidth-10)
	layout.pdf.SetX(x + 5)
	layout.keyValue("Canonical address", layout.model.Subject.CanonicalAddress, layout.bodyWidth-10)
	layout.pdf.SetX(x + 5)
	layout.keyValue("Run ID", layout.model.Run.ID, layout.bodyWidth-10)
	layout.pdf.SetX(x + 5)
	layout.keyValue("Correlation ID", display(layout.model.Run.CorrelationID), layout.bodyWidth-10)
	layout.pdf.SetY(y + 49)

	layout.sectionLabel("Observation window")
	for _, item := range [][2]string{
		{"Started", formatTime(layout.model.Run.StartedAt)},
		{"Completed", formatTime(layout.model.Run.CompletedAt)},
		{"Duration", durationDisplay(layout.model.Run.DurationMilliseconds)},
		{"Run state", string(layout.model.Run.Status)},
	} {
		layout.keyValue(item[0], item[1], layout.bodyWidth)
	}
	layout.pdf.SetY(layout.pdf.GetY() + 3)

	layout.sectionLabel("Producer and collector tools")
	for _, tool := range layout.model.Tools {
		height := 28.0
		if len(tool.Capabilities) > 0 {
			height += layout.textHeight(strings.Join(tool.Capabilities, " · "), layout.bodyWidth-32, 3.8)
		}
		if len(tool.Limitations) > 0 {
			height += layout.textHeight(strings.Join(tool.Limitations, "; "), layout.bodyWidth-32, 3.8)
		}
		if err := layout.ensureSpace(height+4, "Scope and method"); err != nil {
			return err
		}
		y = layout.pdf.GetY()
		layout.card(layout.left, y, layout.bodyWidth, height, layout.colors.softGray, layout.colors.line)
		layout.setFont("B", 10, layout.colors.navy)
		layout.pdf.SetXY(layout.left+5, y+4)
		layout.pdf.CellFormat(layout.bodyWidth-10, 5, tool.Name+"  "+tool.Version, "", 1, "L", false, 0, "")
		layout.setFont("B", 6.8, layout.colors.teal)
		layout.pdf.SetX(layout.left + 5)
		layout.pdf.CellFormat(layout.bodyWidth-10, 4, strings.ToUpper(tool.Role)+" · "+strings.ToUpper(tool.State), "", 1, "L", false, 0, "")
		textY := y + 14
		if len(tool.Capabilities) > 0 {
			textY = layout.writeText(layout.left+5, textY, layout.bodyWidth-10, "Capabilities: "+strings.Join(tool.Capabilities, " · "), "", 7.4, 3.8, layout.colors.ink, "L")
		}
		if len(tool.Limitations) > 0 {
			layout.writeText(layout.left+5, textY+1, layout.bodyWidth-10, "Limitations: "+strings.Join(tool.Limitations, "; "), "", 7.4, 3.8, layout.colors.high, "L")
		}
		layout.pdf.SetY(y + height + 4)
	}

	if err := layout.ensureSpace(25, "Scope and method"); err != nil {
		return err
	}
	layout.sectionLabel("Admitted machine artifacts")
	for _, artifact := range layout.model.Artifacts {
		if err := layout.ensureSpace(22, "Scope and method"); err != nil {
			return err
		}
		validation := artifact.Validations[0]
		y = layout.pdf.GetY()
		layout.card(layout.left, y, layout.bodyWidth, 18, layout.stateFill(string(validation.Status)), layout.colors.line)
		layout.setFont("B", 8.5, layout.colors.navy)
		layout.pdf.SetXY(layout.left+4, y+3)
		layout.pdf.CellFormat(layout.bodyWidth-42, 5, artifact.DisplayName, "", 1, "L", false, 0, "")
		layout.setFont("", 6.8, layout.colors.muted)
		layout.pdf.SetX(layout.left + 4)
		layout.pdf.CellFormat(layout.bodyWidth-42, 4, artifact.Schema+" · "+fmt.Sprintf("%d bytes", artifact.SizeBytes), "", 0, "L", false, 0, "")
		layout.setFont("B", 7.5, layout.colors.teal)
		layout.pdf.SetXY(layout.pageWidth-layout.right-36, y+6)
		layout.pdf.CellFormat(32, 5, strings.ToUpper(string(validation.Status)), "", 0, "R", false, 0, "")
		layout.pdf.SetY(y + 22)
	}
	return nil
}

func (layout *layout) posture() error {
	if err := layout.newPage("Posture axes"); err != nil {
		return err
	}
	layout.pageHeading("No combined score", "Evidence posture axes", "Each axis preserves its authority, confidence, source, time, derivation, and limitation instead of hiding them behind one grade.", assets.IconShieldCheck)
	for index, axis := range layout.model.PostureAxes {
		layout.setFont("", 8, layout.colors.ink)
		labelHeight := layout.textHeight(axis.Label, 53, 4.5)
		valueHeight := layout.textHeight(axis.Value, layout.bodyWidth-75, 4.1)
		contentHeight := max(labelHeight, valueHeight)
		detail := axis.Derivation
		if axis.Limitation != "" {
			if detail != "" {
				detail += " "
			}
			detail += "Limitation: " + axis.Limitation
		}
		detailHeight := 0.0
		if detail != "" {
			detailHeight = layout.textHeight(detail, layout.bodyWidth-16, 3.6) + 2
		}
		height := max(27, 20+contentHeight+detailHeight)
		if err := layout.ensureSpace(height+4, "Posture axes"); err != nil {
			return err
		}
		y := layout.pdf.GetY()
		fill := layout.stateFill(axis.Value)
		layout.card(layout.left, y, layout.bodyWidth, height, fill, layout.colors.line)
		layout.fillColor(layout.colors.teal)
		layout.pdf.RoundedRect(layout.left, y, 3, height, 1.4, "14", "F")
		layout.setFont("B", 7, layout.colors.muted)
		layout.pdf.SetXY(layout.left+7, y+4)
		layout.pdf.CellFormat(48, 4, fmt.Sprintf("AXIS %02d", index+1), "", 1, "L", false, 0, "")
		layout.writeText(layout.left+7, y+9, 53, axis.Label, "B", 9.4, 4.5, layout.colors.navy, "L")
		layout.writeText(layout.left+65, y+9, layout.bodyWidth-72, axis.Value, "B", 9.2, 4.1, layout.colors.blue, "L")
		layout.setFont("", 6.7, layout.colors.muted)
		layout.pdf.SetXY(layout.left+7, y+12+contentHeight)
		layout.pdf.CellFormat(layout.bodyWidth-14, 4, strings.ToUpper(string(axis.Authority))+" · confidence "+strings.ToUpper(display(axis.Confidence))+" · source "+strings.Join(axis.SourceRefs, ", "), "", 1, "L", false, 0, "")
		if detail != "" {
			layout.writeText(layout.left+7, y+17+contentHeight, layout.bodyWidth-14, detail, "", 6.8, 3.6, layout.colors.muted, "L")
		}
		layout.pdf.SetY(y + height + 4)
	}
	return nil
}

func (layout *layout) findings() error {
	if err := layout.newPage("Observed results / findings"); err != nil {
		return err
	}
	layout.pageHeading("Producer interpretations", "Observed results and findings", fmt.Sprintf("%d findings admitted; %d displayed; %d omitted under the explicit collection bound.", layout.model.Findings.TotalCount, len(layout.model.Findings.Items), layout.model.Findings.OmittedCount), assets.IconAlert)
	if len(layout.model.Findings.Items) == 0 {
		return layout.callout("No producer findings", "The producer emitted zero findings. This means no findings were supplied for this run; it does not mean the subject is secure, compliant, or fully assessed.", layout.colors.softAmber, layout.colors.medium, assets.IconAlert)
	}
	for index, finding := range layout.model.Findings.Items {
		if err := layout.renderFinding(index, finding); err != nil {
			return err
		}
	}
	if layout.model.Findings.OmittedCount > 0 {
		return layout.callout("Bounded projection", layout.model.Findings.SelectionRule+". Consult the exact scan JSON digest in the provenance ledger for all findings.", layout.colors.softAmber, layout.colors.medium, assets.IconDownload)
	}
	return nil
}

func (layout *layout) renderFinding(index int, finding evidence.Finding) error {
	metadata := strings.ToUpper(finding.Severity) + " · " + strings.ToUpper(finding.Readiness) + " · confidence " + strings.ToUpper(finding.Confidence) + " · " + strings.ToUpper(string(finding.Authority))
	technical := "Finding ID: " + finding.ID + " · Rule: " + display(finding.RuleRef) + " · Protocol: " + display(finding.Protocol) + " · Algorithm/group: " + display(finding.Algorithm)
	refs := make([]string, 0, len(finding.EvidenceRefs))
	for _, ref := range finding.EvidenceRefs {
		refs = append(refs, ref.ArtifactRef+":"+ref.Pointer)
	}
	body := finding.Description + "\n" + technical + "\nEvidence: " + strings.Join(refs, " · ")
	layout.setFont("", 7.6, layout.colors.ink)
	lines := layout.pdf.SplitText(body, layout.bodyWidth-14)
	if len(lines) == 0 {
		lines = []string{"Not asserted"}
	}
	part := 1
	for len(lines) > 0 {
		available := layout.bottom - layout.pdf.GetY()
		if available < 35 {
			if err := layout.newPage("Observed results / findings"); err != nil {
				return err
			}
			if err := layout.continuationTitle("Observed results / findings"); err != nil {
				return err
			}
			available = layout.bottom - layout.pdf.GetY()
		}
		maximumLines := max(1, int((available-23)/4.0))
		count := min(maximumLines, len(lines))
		height := 21 + float64(count)*4
		y := layout.pdf.GetY()
		fill := layout.stateFill(finding.Readiness)
		layout.card(layout.left, y, layout.bodyWidth, height, fill, layout.colors.line)
		accent := layout.severityColor(finding.Severity)
		layout.fillColor(accent)
		layout.pdf.RoundedRect(layout.left, y, 3, height, 1.4, "14", "F")
		layout.setFont("B", 7, accent)
		layout.pdf.SetXY(layout.left+7, y+3)
		layout.pdf.CellFormat(26, 4, fmt.Sprintf("FINDING %02d", index+1), "", 0, "L", false, 0, "")
		layout.setFont("B", 10, layout.colors.navy)
		title := finding.Title
		if part > 1 {
			title += fmt.Sprintf(" — continued %d", part)
		}
		layout.pdf.SetXY(layout.left+33, y+2.5)
		layout.pdf.CellFormat(layout.bodyWidth-40, 5, truncateRunes(title, 100), "", 1, "L", false, 0, "")
		layout.setFont("B", 6.8, layout.colors.muted)
		layout.pdf.SetXY(layout.left+7, y+9)
		layout.pdf.CellFormat(layout.bodyWidth-14, 4, metadata, "", 1, "L", false, 0, "")
		layout.setFont("", 7.6, layout.colors.ink)
		layout.pdf.SetXY(layout.left+7, y+14)
		layout.pdf.MultiCell(layout.bodyWidth-14, 4, strings.Join(lines[:count], "\n"), "", "L", false)
		layout.pdf.SetY(y + height + 4)
		lines = lines[count:]
		part++
		if len(lines) > 0 {
			if err := layout.newPage("Observed results / findings"); err != nil {
				return err
			}
			if err := layout.continuationTitle("Observed results / findings"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (layout *layout) inventory() error {
	if err := layout.newPage("Cryptographic inventory"); err != nil {
		return err
	}
	layout.pageHeading("CycloneDX CBOM 1.7", "Cryptographic inventory", fmt.Sprintf("%d cryptographic assets admitted; %d displayed; source order does not imply risk priority.", layout.model.Inventory.TotalCount, len(layout.model.Inventory.Items)), assets.IconScan)
	if len(layout.model.Inventory.Items) == 0 {
		return layout.callout("Inventory unavailable", "No cryptographic-asset components were admitted. Inventory remains UNKNOWN; zero rows must not be read as zero cryptographic exposure.", layout.colors.softAmber, layout.colors.medium, assets.IconAlert)
	}
	widths := inventoryWidths(layout.bodyWidth)
	layout.inventoryHeader(widths)
	for index, asset := range layout.model.Inventory.Items {
		if err := layout.inventoryRow(index, asset, widths); err != nil {
			return err
		}
	}
	if layout.model.Inventory.OmittedCount > 0 {
		return layout.callout("Bounded projection", layout.model.Inventory.SelectionRule+". Consult the exact CBOM for all cryptographic assets.", layout.colors.softAmber, layout.colors.medium, assets.IconDownload)
	}
	return nil
}

func inventoryWidths(total float64) []float64 {
	return []float64{total * 0.27, total * 0.15, total * 0.27, total * 0.18, total * 0.13}
}

func (layout *layout) inventoryHeader(widths []float64) {
	x, y := layout.left, layout.pdf.GetY()
	layout.fillColor(layout.colors.navy2)
	layout.pdf.Rect(x, y, layout.bodyWidth, 9, "F")
	labels := []string{"ASSET", "TYPE", "TECHNICAL FACTS", "READINESS", "SOURCE"}
	for index, label := range labels {
		layout.setFont("B", 6.4, layout.colors.white)
		layout.pdf.SetXY(x+2, y+2.3)
		layout.pdf.CellFormat(widths[index]-4, 4, label, "", 0, "L", false, 0, "")
		x += widths[index]
	}
	layout.pdf.SetY(y + 9)
}

func (layout *layout) inventoryRow(index int, asset evidence.InventoryAsset, widths []float64) error {
	technical := strings.Join(nonEmpty([]string{asset.Primitive, asset.Protocol + optionalPrefix(" ", asset.ProtocolVersion), asset.ParameterSet, quantumLevel(asset.NISTQuantumLevel)}), " · ")
	values := []string{asset.Name + "\n" + asset.ID, asset.AssetType, display(technical), display(asset.Readiness), asset.SourceRefs[0] + "\n" + asset.Observation}
	layout.setFont("", 6.7, layout.colors.ink)
	lines := make([][]string, len(values))
	maximumLines := 1
	for column, value := range values {
		lines[column] = layout.pdf.SplitText(value, widths[column]-4)
		if len(lines[column]) == 0 {
			lines[column] = []string{"Not asserted"}
		}
		maximumLines = max(maximumLines, len(lines[column]))
	}
	height := max(11, float64(maximumLines)*3.5+4)
	freshPageCapacity := layout.bottom - (layout.top + 22)
	if layout.pdf.GetY()+height > layout.bottom && height <= freshPageCapacity {
		if err := layout.inventoryContinuation(widths); err != nil {
			return err
		}
	}
	if layout.pdf.GetY()+height <= layout.bottom {
		layout.drawInventoryCells(index, values, widths, height)
		return nil
	}

	for inventoryLinesRemain(lines) {
		available := layout.bottom - layout.pdf.GetY()
		maximumFragmentLines := int((available - 4) / 3.5)
		if maximumFragmentLines < 1 {
			if err := layout.inventoryContinuation(widths); err != nil {
				return err
			}
			continue
		}
		fragments := make([]string, len(lines))
		drawnLines := 1
		for column := range lines {
			count := min(maximumFragmentLines, len(lines[column]))
			fragments[column] = strings.Join(lines[column][:count], "\n")
			lines[column] = lines[column][count:]
			drawnLines = max(drawnLines, count)
		}
		fragmentHeight := max(11, float64(drawnLines)*3.5+4)
		layout.drawInventoryCells(index, fragments, widths, fragmentHeight)
		if inventoryLinesRemain(lines) {
			if err := layout.inventoryContinuation(widths); err != nil {
				return err
			}
		}
	}
	return nil
}

func (layout *layout) inventoryContinuation(widths []float64) error {
	if err := layout.newPage("Cryptographic inventory"); err != nil {
		return err
	}
	if err := layout.continuationTitle("Cryptographic inventory"); err != nil {
		return err
	}
	layout.inventoryHeader(widths)
	return nil
}

func (layout *layout) drawInventoryCells(index int, values []string, widths []float64, height float64) {
	x, y := layout.left, layout.pdf.GetY()
	fill := layout.colors.white
	if index%2 == 1 {
		fill = layout.colors.paper
	}
	layout.fillColor(fill)
	layout.drawColor(layout.colors.line)
	layout.pdf.Rect(x, y, layout.bodyWidth, height, "DF")
	for column, value := range values {
		if column > 0 {
			layout.pdf.Line(x, y, x, y+height)
		}
		style := ""
		textColor := layout.colors.ink
		if column == 0 || column == 3 {
			style = "B"
		}
		if column == 3 {
			textColor = layout.colors.blue
		}
		layout.writeText(x+2, y+2, widths[column]-4, value, style, 6.7, 3.5, textColor, "L")
		x += widths[column]
	}
	layout.pdf.SetY(y + height)
}

func inventoryLinesRemain(lines [][]string) bool {
	for _, column := range lines {
		if len(column) > 0 {
			return true
		}
	}
	return false
}

func (layout *layout) gapsAndLimitations() error {
	if err := layout.newPage("Gaps and limitations"); err != nil {
		return err
	}
	layout.pageHeading("Unknown stays unknown", "Gaps, errors and limitations", "Collection failures, unsupported claims, evidence boundaries, and open questions are first-class report content.", assets.IconAlert)

	layout.sectionLabel("Collection and processing errors")
	if len(layout.model.Errors) == 0 {
		if err := layout.callout("No producer errors admitted", "No typed producer error records were present. This statement applies only to the admitted run and is not a completeness claim.", layout.colors.softGreen, layout.colors.teal, assets.IconShieldCheck); err != nil {
			return err
		}
	} else {
		for _, item := range layout.model.Errors {
			if err := layout.callout(item.Code, item.Description+" Stage: "+item.Stage+". Source: "+item.SourceRef+".", layout.colors.softRed, layout.colors.high, assets.IconAlert); err != nil {
				return err
			}
		}
	}

	if err := layout.ensureSpace(20, "Gaps and limitations"); err != nil {
		return err
	}
	layout.sectionLabel("Declared limitations")
	for index, limitation := range layout.model.Limitations {
		layout.setFont("", 8, layout.colors.ink)
		height := max(16, layout.textHeight(limitation.Description, layout.bodyWidth-22, 4)+8)
		if err := layout.ensureSpace(height+3, "Gaps and limitations"); err != nil {
			return err
		}
		y := layout.pdf.GetY()
		layout.card(layout.left, y, layout.bodyWidth, height, layout.colors.softGray, layout.colors.line)
		layout.fillColor(layout.severityColor(limitation.Severity))
		layout.pdf.RoundedRect(layout.left+4, y+4, 8, 8, 4, "1234", "F")
		layout.setFont("B", 7, layout.colors.white)
		layout.pdf.SetXY(layout.left+4, y+5.5)
		layout.pdf.CellFormat(8, 4, fmt.Sprintf("%d", index+1), "", 0, "C", false, 0, "")
		layout.writeText(layout.left+17, y+4, layout.bodyWidth-22, limitation.Description, "", 8, 4, layout.colors.ink, "L")
		layout.pdf.SetY(y + height + 3)
	}

	if err := layout.ensureSpace(20, "Gaps and limitations"); err != nil {
		return err
	}
	layout.sectionLabel("Unresolved questions")
	for index, question := range layout.model.UnresolvedQuestions {
		layout.setFont("", 8, layout.colors.ink)
		height := max(11, layout.textHeight(question, layout.bodyWidth-16, 4)+5)
		if err := layout.ensureSpace(height+2, "Gaps and limitations"); err != nil {
			return err
		}
		y := layout.pdf.GetY()
		layout.setFont("B", 8, layout.colors.teal)
		layout.pdf.SetXY(layout.left, y)
		layout.pdf.CellFormat(8, 4, fmt.Sprintf("Q%d", index+1), "", 0, "L", false, 0, "")
		layout.writeText(layout.left+10, y, layout.bodyWidth-10, question, "", 8, 4, layout.colors.ink, "L")
		layout.pdf.SetY(y + height + 2)
	}
	return nil
}

func (layout *layout) provenance() error {
	if err := layout.newPage("Provenance and integrity"); err != nil {
		return err
	}
	layout.pageHeading("Exact-byte ledger", "Provenance and report integrity", "The source artifacts, model, renderer, fonts, and visual assets that produced this bounded human-readable projection.", assets.IconDownload)

	layout.sectionLabel("Input artifact hash ledger")
	for _, artifact := range layout.model.Artifacts {
		layout.setFont("", 7.2, layout.colors.ink)
		digestHeight := layout.textHeight(artifact.Digest.Value, layout.bodyWidth-43, 3.7)
		height := max(38, 28+digestHeight)
		if err := layout.ensureSpace(height+4, "Provenance and integrity"); err != nil {
			return err
		}
		y := layout.pdf.GetY()
		layout.card(layout.left, y, layout.bodyWidth, height, layout.colors.softBlue, layout.colors.line)
		layout.setFont("B", 9.2, layout.colors.navy)
		layout.pdf.SetXY(layout.left+5, y+4)
		layout.pdf.CellFormat(layout.bodyWidth-10, 5, artifact.DisplayName, "", 1, "L", false, 0, "")
		layout.setFont("", 6.8, layout.colors.muted)
		layout.pdf.SetX(layout.left + 5)
		layout.pdf.CellFormat(layout.bodyWidth-10, 4, artifact.Role+" · "+artifact.MediaType+" · "+artifact.Schema+" · "+fmt.Sprintf("%d bytes", artifact.SizeBytes), "", 1, "L", false, 0, "")
		layout.setFont("B", 6.8, layout.colors.teal)
		layout.pdf.SetXY(layout.left+5, y+16)
		layout.pdf.CellFormat(36, 4, "SHA-256", "", 0, "L", false, 0, "")
		layout.writeText(layout.left+42, y+16, layout.bodyWidth-47, artifact.Digest.Value, "", 7.2, 3.7, layout.colors.ink, "L")
		layout.setFont("", 6.6, layout.colors.muted)
		layout.pdf.SetXY(layout.left+5, y+25+digestHeight)
		layout.pdf.CellFormat(layout.bodyWidth-10, 4, "Produced "+formatTime(artifact.ProducedAt)+" · validation "+string(artifact.Validations[0].Status)+" · "+artifact.Relationship, "", 0, "L", false, 0, "")
		layout.pdf.SetY(y + height + 4)
	}

	if err := layout.ensureSpace(24, "Provenance and integrity"); err != nil {
		return err
	}
	layout.sectionLabel("Report compiler identity")
	for _, item := range [][2]string{
		{"Report schema", layout.model.SchemaVersion},
		{"Report model SHA-256", layout.modelDigest},
		{"Generator", "breachsafe-pdf " + layout.renderer.generatorVersion + optionalPrefix(" @ ", layout.renderer.generatorCommit)},
		{"PDF renderer", rendererName + " " + libraryVersion()},
		{"Font bundle SHA-256", fontDigest()},
		{"Asset bundle SHA-256", layout.modelAssetDigest()},
	} {
		if err := layout.ensureSpace(9, "Provenance and integrity"); err != nil {
			return err
		}
		layout.keyValue(item[0], item[1], layout.bodyWidth)
	}

	if err := layout.ensureSpace(24, "Provenance and integrity"); err != nil {
		return err
	}
	layout.sectionLabel("Renderer capability record")
	for _, capability := range capabilityResults() {
		layout.setFont("B", 7.2, layout.colors.ink)
		nameHeight := layout.textHeight(capability.Name, 54, 3.5)
		layout.setFont("", 6.7, layout.colors.muted)
		detailHeight := layout.textHeight(capability.Detail, layout.bodyWidth-89, 3.5)
		rowHeight := max(8, max(nameHeight, detailHeight)+2)
		if err := layout.ensureSpace(rowHeight, "Provenance and integrity"); err != nil {
			return err
		}
		y := layout.pdf.GetY()
		statusColor := layout.colors.teal
		if capability.Status != "pass" {
			statusColor = layout.colors.medium
		}
		layout.writeText(layout.left, y, 54, capability.Name, "B", 7.2, 3.5, layout.colors.ink, "L")
		layout.writeText(layout.left+57, y, 29, strings.ToUpper(capability.Status), "B", 6.7, 3.5, statusColor, "L")
		layout.writeText(layout.left+89, y, layout.bodyWidth-89, capability.Detail, "", 6.7, 3.5, layout.colors.muted, "L")
		layout.pdf.SetY(y + rowHeight)
	}

	if err := layout.newPage("Integrity receipt"); err != nil {
		return err
	}
	layout.pageHeading("Verification boundary", "External integrity receipt", "How to bind this human-readable projection to the exact final bytes and the admitted machine evidence.", assets.IconDownload)
	if err := layout.callout("Final PDF SHA-256 is external", "The PDF cannot embed its own final digest without changing the bytes being hashed. The adjacent RenderResult JSON records the exact final PDF SHA-256, byte count, page count, model digest, render-request digest, and component identities.", layout.colors.softAmber, layout.colors.medium, assets.IconDownload); err != nil {
		return err
	}
	layout.sectionLabel("Verification procedure")
	steps := []string{
		"Compute SHA-256 over the exact PDF bytes delivered to the consumer.",
		"Compare the digest, byte count, and page count with the adjacent RenderResult JSON.",
		"Compare the CBOM and scan JSON digests with the exact source files retained or packaged with the report.",
		"If carried in an ePack, verify that package independently and confirm it contains these unchanged PDF and source bytes.",
	}
	for index, step := range steps {
		y := layout.pdf.GetY()
		layout.fillColor(layout.colors.navy2)
		layout.pdf.RoundedRect(layout.left, y, 9, 9, 4.5, "1234", "F")
		layout.setFont("B", 7, layout.colors.white)
		layout.pdf.SetXY(layout.left, y+2)
		layout.pdf.CellFormat(9, 4, fmt.Sprintf("%d", index+1), "", 0, "C", false, 0, "")
		textHeight := layout.textHeight(step, layout.bodyWidth-15, 4)
		layout.writeText(layout.left+14, y, layout.bodyWidth-14, step, "", 8, 4, layout.colors.ink, "L")
		layout.pdf.SetY(y + max(11, textHeight+3))
	}
	layout.pdf.SetY(layout.pdf.GetY() + 4)
	return layout.callout("Integrity is not authenticity", layout.model.IntegrityStatement, layout.colors.softBlue, layout.colors.blue, assets.IconShieldCheck)
}

func (layout *layout) modelAssetDigest() string {
	return assets.BundleDigest()
}

func highestSeverity(findings []evidence.Finding) string {
	if len(findings) == 0 {
		return "none asserted"
	}
	ranks := map[string]int{"info": 1, "low": 2, "medium": 3, "high": 4, "critical": 5}
	result := "info"
	for _, finding := range findings {
		if ranks[strings.ToLower(finding.Severity)] > ranks[result] {
			result = strings.ToLower(finding.Severity)
		}
	}
	return result
}

func severityCounts(findings []evidence.Finding) map[string]int {
	result := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0}
	for _, finding := range findings {
		result[strings.ToLower(finding.Severity)]++
	}
	return result
}

func selectedCoverAxes(axes []evidence.PostureAxis) []evidence.PostureAxis {
	result := make([]evidence.PostureAxis, 0, 3)
	for _, wanted := range []string{"quantum-readiness", "key-establishment", "coverage-quality"} {
		for _, axis := range axes {
			if axis.ID == wanted {
				result = append(result, axis)
				break
			}
		}
	}
	return result
}

func summaryReadiness(model evidence.CommunitySingleScan) string {
	for _, axis := range model.PostureAxes {
		if axis.ID == "quantum-readiness" {
			return axis.Value
		}
	}
	return "unknown"
}

func durationDisplay(milliseconds int64) string {
	if milliseconds < 1_000 {
		return fmt.Sprintf("%d ms", milliseconds)
	}
	return fmt.Sprintf("%.2f s", float64(milliseconds)/1_000)
}

func correlationSeverity(state string) string {
	switch state {
	case "matched":
		return "info"
	case "mismatched":
		return "high"
	default:
		return "medium"
	}
}

func quantumLevel(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("NIST QL%d", *value)
}

func optionalPrefix(prefix, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return prefix + value
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}
