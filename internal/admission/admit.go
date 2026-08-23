// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package admission

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
)

const (
	maxDisplayedFindings  = 250
	maxDisplayedInventory = 1_000
)

// Admit validates exact producer bytes, correlates the sources, and constructs
// the fixed Community Single-Scan report model. It performs no file or network
// access and does not mutate the supplied byte slices.
func Admit(ctx context.Context, requestBytes, cbomBytes, scanJSONBytes []byte, limits Limits) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, fault.Wrap(fault.CodeCanceled, "admission.admit", err)
	}
	for _, input := range []struct {
		name    string
		data    []byte
		maximum int
	}{
		{"request", requestBytes, limits.MaxRequestBytes},
		{"cbom", cbomBytes, limits.MaxCBOMBytes},
		{"scan_json", scanJSONBytes, limits.MaxScanJSONBytes},
	} {
		if len(input.data) == 0 {
			return Result{}, fault.New(fault.CodeInvalidInput, "admission.admit", input.name, "input is required")
		}
		if len(input.data) > input.maximum {
			return Result{}, fault.Format(fault.CodeLimitExceeded, "admission.admit", input.name, "got %d bytes; maximum is %d", len(input.data), input.maximum)
		}
	}

	var request Request
	if err := decodeJSON(requestBytes, &request, true, "admission.request_decode"); err != nil {
		return Result{}, err
	}
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}

	if err := rejectUnknownTopLevel(scanJSONBytes, scanTopLevelFields, "admission.scan_decode"); err != nil {
		return Result{}, err
	}
	var scan qureddyDocument
	if err := decodeJSON(scanJSONBytes, &scan, false, "admission.scan_decode"); err != nil {
		return Result{}, err
	}
	if scan.SchemaVersion != request.ScanJSON.Schema {
		return Result{}, fault.New(fault.CodeSchemaMismatch, "admission.scan_decode", "schema_version", "declared request schema differs from the exact scan JSON bytes")
	}
	if err := validateQuReddy(ctx, scan, limits); err != nil {
		return Result{}, err
	}

	if err := rejectUnknownTopLevel(cbomBytes, cbomTopLevelFields, "admission.cbom_decode"); err != nil {
		return Result{}, err
	}
	var cbom cycloneDXDocument
	if err := decodeJSON(cbomBytes, &cbom, false, "admission.cbom_decode"); err != nil {
		return Result{}, err
	}
	if "cyclonedx:"+cbom.SpecVersion != request.CBOM.Schema {
		return Result{}, fault.New(fault.CodeSchemaMismatch, "admission.cbom_decode", "specVersion", "declared request schema differs from the exact CBOM bytes")
	}
	cbomProperties, err := validateCycloneDX(ctx, cbom, limits)
	if err != nil {
		return Result{}, err
	}

	requestDigest := evidence.DigestBytes(requestBytes)
	cbomDigest := evidence.DigestBytes(cbomBytes)
	scanDigest := evidence.DigestBytes(scanJSONBytes)
	if err := compareExpectedDigest("cbom", request.CBOM.ExpectedSHA256, cbomDigest); err != nil {
		return Result{}, err
	}
	if err := compareExpectedDigest("scan_json", request.ScanJSON.ExpectedSHA256, scanDigest); err != nil {
		return Result{}, err
	}

	correlation := correlate(scan, cbom, cbomProperties)
	if correlation.State == "mismatched" && !request.AllowCorrelationMismatch {
		return Result{}, fault.New(fault.CodeCorrelationMismatch, "admission.correlate", "sources", correlation.Basis)
	}

	model, err := buildModel(request, scan, cbom, correlation, cbomDigest, scanDigest, len(cbomBytes), len(scanJSONBytes))
	if err != nil {
		return Result{}, err
	}
	model = evidence.Canonicalize(model)
	if err := evidence.Validate(ctx, model, limits.Model); err != nil {
		return Result{}, err
	}
	return Result{
		Model:               model,
		RequestBytesSHA256:  requestDigest,
		CBOMBytesSHA256:     cbomDigest,
		ScanJSONBytesSHA256: scanDigest,
	}, nil
}

func compareExpectedDigest(field, expected, actual string) error {
	if expected != "" && expected != actual {
		return fault.New(fault.CodeDigestMismatch, "admission.digest", field, "exact input bytes do not match expected SHA-256")
	}
	return nil
}

func correlate(scan qureddyDocument, cbom cycloneDXDocument, properties map[string]string) evidence.Correlation {
	checks := []correlationCheck{
		{"scan ID", scan.Scan.ScanID, properties["qureddy:scan.id"]},
		{"target", scan.Target.Locator, properties["qureddy:target.locator"]},
		{"scanner", scan.Scan.ScannerName, properties["qureddy:scan.scanner_name"]},
		{"run status", scan.Scan.Status, properties["qureddy:scan.status"]},
		{"readiness", scan.Summary.Readiness, properties["qureddy:scan.readiness"]},
		{"finding count", strconv.Itoa(len(scan.Findings)), properties["qureddy:scan.finding_count"]},
		{"attempt count", strconv.Itoa(scan.Scan.TotalAttempts), properties["qureddy:scan.total_attempts"]},
	}
	if value := properties["qureddy:scan.started_at"]; value != "" {
		checks = append(checks, correlationCheck{"start time", scan.Scan.StartedAt.UTC().Format(time.RFC3339Nano), canonicalTime(value)})
	}
	if value := properties["qureddy:scan.completed_at"]; value != "" {
		checks = append(checks, correlationCheck{"completion time", scan.Scan.CompletedAt.UTC().Format(time.RFC3339Nano), canonicalTime(value)})
	}
	if version := toolVersion(cbom, "qureddy"); version != "" {
		checks = append(checks, correlationCheck{"QuReddy version", scan.Scan.ScannerVersion, version})
	}

	missing := make([]string, 0)
	mismatched := make([]string, 0)
	matched := 0
	for _, check := range checks {
		if strings.TrimSpace(check.right) == "" {
			missing = append(missing, check.name)
			continue
		}
		if check.left != check.right {
			mismatched = append(mismatched, check.name)
			continue
		}
		matched++
	}
	refs := []string{"cbom", "scan-json"}
	if len(mismatched) > 0 {
		return evidence.Correlation{
			State:      "mismatched",
			Basis:      "Conflicting same-run fields: " + strings.Join(mismatched, ", ") + ". Sources were not silently merged.",
			SourceRefs: refs,
		}
	}
	if len(missing) > 0 {
		return evidence.Correlation{
			State:      "unknown",
			Basis:      fmt.Sprintf("%d fields matched, but explicit correlation fields are absent: %s.", matched, strings.Join(missing, ", ")),
			SourceRefs: refs,
		}
	}
	return evidence.Correlation{
		State:      "matched",
		Basis:      fmt.Sprintf("Matched %d explicit fields across exact CBOM and scan JSON bytes: scan, subject, state, counts, times, and producer version.", matched),
		SourceRefs: refs,
	}
}

type correlationCheck struct {
	name  string
	left  string
	right string
}

func canonicalTime(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.UTC().Format(time.RFC3339Nano)
}

func toolVersion(cbom cycloneDXDocument, name string) string {
	for _, component := range cbom.Metadata.Tools.Components {
		if strings.EqualFold(component.Name, name) {
			return component.Version
		}
	}
	return ""
}

func buildModel(request Request, scan qureddyDocument, cbom cycloneDXDocument, correlation evidence.Correlation, cbomDigest, scanDigest string, cbomSize, scanSize int) (evidence.CommunitySingleScan, error) {
	tools := buildTools(scan, cbom)
	artifacts := []evidence.Artifact{
		{
			ID:                    "cbom",
			Role:                  "inventory/cbom",
			DisplayName:           "CycloneDX Cryptographic Bill of Materials",
			MediaType:             "application/vnd.cyclonedx+json",
			Schema:                "https://cyclonedx.org/schema/bom-1.7.schema.json",
			DeclaredSchemaVersion: cbom.SpecVersion,
			Digest:                evidence.Digest{Algorithm: "sha256", Value: cbomDigest},
			SizeBytes:             cbomSize,
			ProducerToolRef:       "tool-qureddy",
			ProducedAt:            cbom.Metadata.Timestamp.UTC(),
			Validations: []evidence.ValidationResult{{
				Validator: "breachsafe-pdf/admission", Version: "v1alpha1", Status: evidence.ValidationValid,
				Detail: "CycloneDX 1.7 structure, CBOM semantics, references, counts, and QuReddy metadata accepted",
			}},
			Relationship: "correlated with scan-json: " + correlation.State,
		},
		{
			ID:                    "scan-json",
			Role:                  "scan/result",
			DisplayName:           "QuReddy producer-native scan result",
			MediaType:             "application/json",
			Schema:                "qureddy.scan.v1",
			DeclaredSchemaVersion: scan.SchemaVersion,
			Digest:                evidence.Digest{Algorithm: "sha256", Value: scanDigest},
			SizeBytes:             scanSize,
			ProducerToolRef:       "tool-qureddy",
			ProducedAt:            scan.Scan.CompletedAt.UTC(),
			Validations: []evidence.ValidationResult{{
				Validator: "breachsafe-pdf/admission", Version: "v1alpha1", Status: evidence.ValidationValid,
				Detail: "qureddy.scan.v1 structure, enums, stable IDs, references, and deterministic summary counts accepted",
			}},
			Relationship: "correlated with cbom: " + correlation.State,
		},
	}

	coverage := buildCoverage(scan)
	findings := buildFindings(scan, cbom)
	inventory, err := buildInventory(cbom)
	if err != nil {
		return evidence.CommunitySingleScan{}, err
	}
	limitations := []evidence.Limitation{
		{ID: "single-run-boundary", Severity: "info", Description: "This Community report covers one producer run and one primary subject; it is not a fleet or organization assessment.", SourceRefs: []string{"scan-json"}},
		{ID: "not-certification", Severity: "info", Description: "This evidence-backed report is not a certification, compliance determination, or attestation.", SourceRefs: []string{"cbom", "scan-json"}},
		{ID: "source-authority", Severity: "info", Description: "The PDF is a bounded human-readable projection. The exact CBOM and scan JSON bytes remain authoritative.", SourceRefs: []string{"cbom", "scan-json"}},
	}
	if correlation.State != "matched" {
		limitations = append(limitations, evidence.Limitation{ID: "source-correlation-" + correlation.State, Severity: "high", Description: correlation.Basis, SourceRefs: []string{"cbom", "scan-json"}})
	}
	if scan.Scan.Status != "completed" {
		limitations = append(limitations, evidence.Limitation{ID: "scan-not-completed", Severity: "high", Description: "The producer run did not complete: " + scan.Scan.Status + ". Missing evidence remains UNKNOWN.", SourceRefs: []string{"scan-json"}})
	}
	if findings.OmittedCount > 0 {
		limitations = append(limitations, evidence.Limitation{ID: "finding-display-bound", Severity: "medium", Description: fmt.Sprintf("The PDF displays %d of %d findings using the documented deterministic selection rule; the exact scan JSON contains all rows.", len(findings.Items), findings.TotalCount), SourceRefs: []string{"scan-json"}})
	}
	if inventory.OmittedCount > 0 {
		limitations = append(limitations, evidence.Limitation{ID: "inventory-display-bound", Severity: "medium", Description: fmt.Sprintf("The PDF displays %d of %d CBOM assets using the documented deterministic selection rule; the exact CBOM contains all rows.", len(inventory.Items), inventory.TotalCount), SourceRefs: []string{"cbom"}})
	}

	unresolved := []string{"What additional systems, ports, protocols, and observation windows exist outside this single declared target and run?"}
	if scan.Summary.Readiness == "unknown" || scan.Scan.Status != "completed" {
		unresolved = append(unresolved, "What collection capability or evidence is required to resolve the producer's UNKNOWN readiness state?")
	}

	duration := scan.Scan.CompletedAt.Sub(scan.Scan.StartedAt).Milliseconds()
	model := evidence.CommunitySingleScan{
		SchemaVersion: evidence.SchemaVersion,
		Identity:      request.Identity,
		Run: evidence.Run{
			ID: scan.Scan.ScanID, CorrelationID: cbom.SerialNumber, Status: mapRunStatus(scan),
			StartedAt: scan.Scan.StartedAt.UTC(), CompletedAt: scan.Scan.CompletedAt.UTC(),
			DurationMilliseconds: duration, Attempt: 1,
		},
		Subject: evidence.Subject{
			ID: "subject-primary", Kind: scan.Target.Scheme + ".endpoint", DisplayName: subjectDisplayName(scan),
			CanonicalAddress: scan.Target.Locator, SourceRefs: []string{"cbom", "scan-json"},
		},
		Tools:               tools,
		Artifacts:           artifacts,
		Correlation:         correlation,
		Coverage:            coverage,
		PostureAxes:         buildPostureAxes(scan, cbom, coverage),
		Findings:            findings,
		Inventory:           inventory,
		Errors:              buildErrors(scan),
		Limitations:         limitations,
		UnresolvedQuestions: unresolved,
		IntegrityStatement:  "SHA-256 digests identify the exact input bytes supplied to this report. They do not prove semantic correlation, completeness, authenticity, compliance, or freshness. The final PDF digest is written only to the adjacent result JSON after rendering.",
		RenderOptions:       request.RenderOptions,
	}
	return model, nil
}

func buildTools(scan qureddyDocument, cbom cycloneDXDocument) []evidence.Tool {
	tools := []evidence.Tool{{
		ID: "tool-qureddy", Name: "QuReddy", Version: scan.Scan.ScannerVersion, Role: "producer/scanner",
		Capabilities: []string{scan.Scan.ScannerName + " single-target scan", "qureddy.scan.v1", "CycloneDX 1.7 CBOM"}, State: toolState(scan.Scan.Status),
	}}
	seen := map[string]struct{}{"qureddy": {}}
	for _, component := range cbom.Metadata.Tools.Components {
		name := strings.ToLower(component.Name)
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		state := "available"
		limitations := []string(nil)
		capabilities := []string(nil)
		for _, dependency := range scan.Dependencies {
			if !strings.EqualFold(dependency.Name, component.Name) {
				continue
			}
			if dependency.SupportsTLS13Groups {
				capabilities = append(capabilities, "TLS 1.3 group enumeration")
			}
			if dependency.SupportsX25519MLKEM768 {
				capabilities = append(capabilities, "X25519MLKEM768 probe")
			}
			if dependency.FailureCategory != nil {
				state = "unavailable"
				limitations = append(limitations, *dependency.FailureCategory)
			}
		}
		tools = append(tools, evidence.Tool{
			ID: "tool-" + safeID(component.Name), Name: component.Name, Version: displayUnknown(component.Version),
			Role: "local collector dependency", Capabilities: capabilities, State: state, Limitations: limitations,
		})
	}
	return tools
}

func buildCoverage(scan qureddyDocument) evidence.Coverage {
	coverage := evidence.Coverage{
		Status: evidence.CoveragePartial, Requested: 1, Attempted: 0, Completed: 0,
		Authority: evidence.AuthorityInferred, SourceRefs: []string{"scan-json"},
		Derivation: "One explicitly requested target; attempted from producer total_attempts; completed only when scan.status=completed.",
	}
	if scan.Scan.TotalAttempts > 0 {
		coverage.Attempted = 1
	}
	if scan.Scan.Status == "completed" {
		coverage.Status = evidence.CoverageComplete
		coverage.Completed = 1
	} else if scan.Scan.TotalAttempts == 0 {
		coverage.Status = evidence.CoverageUnavailable
	}
	if coverage.Completed == 0 {
		coverage.NotAssessed = []string{scan.Target.Locator + " did not produce a completed scan"}
	}
	return coverage
}

func buildPostureAxes(scan qureddyDocument, cbom cycloneDXDocument, coverage evidence.Coverage) []evidence.PostureAxis {
	observedGroups := make([]string, 0)
	protocols := make([]string, 0)
	confidences := make([]string, 0, len(scan.Findings))
	for _, item := range scan.Evidence {
		if item.NegotiatedGroup != nil && *item.NegotiatedGroup != "" {
			observedGroups = appendUnique(observedGroups, *item.NegotiatedGroup)
		}
		if item.ProtocolVersion != nil && *item.ProtocolVersion != "" {
			protocols = appendUnique(protocols, *item.ProtocolVersion)
		}
	}
	for _, finding := range scan.Findings {
		confidences = append(confidences, finding.Confidence)
	}
	keyValue := "UNKNOWN — no negotiated/offered group asserted"
	keyAuthority := evidence.AuthorityUnknown
	if len(observedGroups) > 0 {
		keyValue = strings.Join(observedGroups, ", ")
		keyAuthority = evidence.AuthorityObserved
	}
	protocolValue := strings.ToUpper(scan.Target.Scheme)
	if len(protocols) > 0 {
		protocolValue += " " + strings.Join(protocols, ", ")
	}
	classicalValue := "UNKNOWN — not independently scored"
	if scan.Summary.Readiness == "classically_weak" {
		classicalValue = "classically_weak (producer assertion)"
	}
	confidenceValue := aggregateConfidence(confidences)
	return []evidence.PostureAxis{
		{ID: "quantum-readiness", Label: "Quantum readiness", Value: scan.Summary.Readiness, Authority: evidence.AuthorityToolAsserted, Confidence: confidenceValue, SourceRefs: []string{"cbom", "scan-json"}, ObservedAt: scan.Scan.CompletedAt.UTC(), Derivation: "QuReddy summary readiness cross-checked against CBOM metadata."},
		{ID: "key-establishment", Label: "Key establishment observations", Value: keyValue, Authority: keyAuthority, Confidence: confidenceValue, SourceRefs: []string{"scan-json"}, ObservedAt: scan.Scan.CompletedAt.UTC()},
		{ID: "classical-hygiene", Label: "Classical hygiene", Value: classicalValue, Authority: authorityForKnown(classicalValue), Confidence: confidenceValue, SourceRefs: []string{"scan-json"}, ObservedAt: scan.Scan.CompletedAt.UTC(), Limitation: "No combined or independent classical security score was inferred."},
		{ID: "protocol-posture", Label: "Protocol posture", Value: protocolValue, Authority: evidence.AuthorityObserved, Confidence: confidenceValue, SourceRefs: []string{"cbom", "scan-json"}, ObservedAt: scan.Scan.CompletedAt.UTC()},
		{ID: "coverage-quality", Label: "Coverage quality", Value: string(coverage.Status), Authority: evidence.AuthorityInferred, Confidence: "high", SourceRefs: []string{"scan-json"}, ObservedAt: scan.Scan.CompletedAt.UTC(), Derivation: coverage.Derivation},
		{ID: "crypto-inventory", Label: "CBOM inventory", Value: fmt.Sprintf("%d cryptographic assets admitted", len(cbom.Components)), Authority: evidence.AuthorityImported, Confidence: "high", SourceRefs: []string{"cbom"}, ObservedAt: cbom.Metadata.Timestamp.UTC()},
	}
}

func buildFindings(scan qureddyDocument, cbom cycloneDXDocument) evidence.FindingCollection {
	maximum := min(len(scan.Findings), maxDisplayedFindings)
	items := make([]evidence.Finding, 0, maximum)
	evidenceIndexes := make(map[string]int, len(scan.Evidence))
	for index, item := range scan.Evidence {
		evidenceIndexes[item.ID] = index
	}
	cbomRules := cbomRulePointers(cbom)
	for index, finding := range scan.Findings[:maximum] {
		refs := []evidence.EvidenceRef{{ArtifactRef: "scan-json", Pointer: fmt.Sprintf("/findings/%d", index), Description: "QuReddy producer finding"}}
		for _, evidenceID := range finding.EvidenceIDs {
			if evidenceIndex, ok := evidenceIndexes[evidenceID]; ok {
				refs = append(refs, evidence.EvidenceRef{ArtifactRef: "scan-json", Pointer: fmt.Sprintf("/evidence/%d", evidenceIndex), Description: "Supporting producer observation " + evidenceID})
			}
		}
		sources := []string{"scan-json"}
		if pointer, ok := cbomRules[finding.RuleID]; ok {
			refs = append(refs, evidence.EvidenceRef{ArtifactRef: "cbom", Pointer: pointer, Description: "Correlated CBOM asset or annotation"})
			sources = append(sources, "cbom")
		}
		algorithm := firstNonEmpty(pointerValue(finding.Algorithm), pointerValue(finding.NegotiatedGroup), pointerValue(finding.ParameterSetIdentifier))
		items = append(items, evidence.Finding{
			ID: finding.ID, Title: finding.Title, Description: finding.Description, Outcome: "unknown",
			Severity: finding.Severity, Readiness: finding.Readiness, Confidence: finding.Confidence,
			Authority: evidence.AuthorityToolAsserted, SubjectRef: "subject-primary", SourceRefs: sources,
			EvidenceRefs: refs, RuleRef: finding.RuleID, ObservedAt: scan.Scan.CompletedAt.UTC(),
			Protocol: finding.Protocol, Algorithm: algorithm,
		})
	}
	omitted := len(scan.Findings) - len(items)
	collection := evidence.FindingCollection{TotalCount: len(scan.Findings), Items: items, OmittedCount: omitted}
	if omitted > 0 {
		collection.SelectionRule = "source order, first 250 rows; exact source retains all findings"
		collection.AuthoritativeArtifactRef = "scan-json"
	}
	return collection
}

func cbomRulePointers(cbom cycloneDXDocument) map[string]string {
	result := make(map[string]string)
	for index, component := range cbom.Components {
		properties, err := propertyMap(component.Properties, "component")
		if err == nil && properties["qureddy:rule_id"] != "" {
			result[properties["qureddy:rule_id"]] = fmt.Sprintf("/components/%d", index)
		}
	}
	return result
}

func buildInventory(cbom cycloneDXDocument) (evidence.InventoryCollection, error) {
	maximum := min(len(cbom.Components), maxDisplayedInventory)
	items := make([]evidence.InventoryAsset, 0, maximum)
	for index, component := range cbom.Components[:maximum] {
		properties, err := propertyMap(component.Properties, "component")
		if err != nil {
			return evidence.InventoryCollection{}, err
		}
		asset := evidence.InventoryAsset{
			ID: component.BomRef, Name: component.Name, AssetType: component.CryptoProperties.AssetType,
			Observation: displayUnknown(properties["qureddy:observation"]), Readiness: displayUnknown(properties["qureddy:readiness"]),
			Severity: properties["qureddy:severity"], Authority: evidence.AuthorityImported,
			SubjectRef: "subject-primary", SourceRefs: []string{"cbom"},
			EvidenceRefs: []evidence.EvidenceRef{{ArtifactRef: "cbom", Pointer: fmt.Sprintf("/components/%d", index), Description: "CycloneDX cryptographic asset"}},
		}
		if algorithm := component.CryptoProperties.AlgorithmProperties; algorithm != nil {
			asset.Primitive = algorithm.Primitive
			asset.ParameterSet = firstNonEmpty(algorithm.ParameterSetIdentifier, algorithm.Curve)
			asset.NISTQuantumLevel = algorithm.NISTQuantumSecurityLevel
		}
		if protocol := component.CryptoProperties.ProtocolProperties; protocol != nil {
			asset.Protocol = protocol.Type
			asset.ProtocolVersion = protocol.Version
		}
		items = append(items, asset)
	}
	omitted := len(cbom.Components) - len(items)
	collection := evidence.InventoryCollection{TotalCount: len(cbom.Components), Items: items, OmittedCount: omitted}
	if omitted > 0 {
		collection.SelectionRule = "source order, first 1000 rows; exact source retains all cryptographic assets"
		collection.AuthoritativeArtifactRef = "cbom"
	}
	return collection, nil
}

func buildErrors(scan qureddyDocument) []evidence.ErrorRecord {
	errors := make([]evidence.ErrorRecord, 0)
	if scan.Scan.Status != "completed" {
		errors = append(errors, evidence.ErrorRecord{ID: "scan-status", Stage: "collection", SourceRef: "scan-json", Code: scan.Scan.Status, Description: "Producer reported a non-completed scan state.", SubjectRef: "subject-primary"})
	}
	for index, dependency := range scan.Dependencies {
		if dependency.FailureCategory != nil {
			errors = append(errors, evidence.ErrorRecord{ID: fmt.Sprintf("dependency-%d", index+1), Stage: "collector-capability", SourceRef: "scan-json", Code: *dependency.FailureCategory, Description: dependency.Name + " was unavailable or lacked a required capability.", SubjectRef: "subject-primary"})
		}
	}
	for index, item := range scan.Evidence {
		if item.FailureCategory != nil {
			errors = append(errors, evidence.ErrorRecord{ID: fmt.Sprintf("evidence-%d", index+1), Stage: "observation", SourceRef: "scan-json", Code: *item.FailureCategory, Description: "A producer observation attempt did not establish the requested fact.", SubjectRef: "subject-primary", EvidenceRefs: []string{item.ID}})
		}
	}
	return errors
}

func mapRunStatus(scan qureddyDocument) evidence.RunStatus {
	if scan.Scan.Status == "completed" {
		if len(scan.Findings) == 0 {
			return evidence.RunCompletedEmpty
		}
		return evidence.RunCompleted
	}
	if strings.HasPrefix(scan.Scan.Status, "local_") {
		return evidence.RunUnavailable
	}
	return evidence.RunFailed
}

func subjectDisplayName(scan qureddyDocument) string {
	if len(scan.Assets) > 0 && strings.TrimSpace(scan.Assets[0].DisplayName) != "" {
		return scan.Assets[0].DisplayName
	}
	return scan.Target.Host + ":" + strconv.Itoa(scan.Target.Port)
}

func toolState(scanStatus string) string {
	if scanStatus == "completed" {
		return "completed"
	}
	if strings.HasPrefix(scanStatus, "local_") {
		return "unavailable"
	}
	return "failed"
}

func authorityForKnown(value string) evidence.Authority {
	if strings.HasPrefix(value, "UNKNOWN") {
		return evidence.AuthorityUnknown
	}
	return evidence.AuthorityToolAsserted
}

func aggregateConfidence(values []string) string {
	if len(values) == 0 {
		return "unknown"
	}
	rank := map[string]int{"low": 1, "medium": 2, "high": 3}
	minimum := 4
	result := "unknown"
	for _, value := range values {
		if rank[value] < minimum {
			minimum = rank[value]
			result = value
		}
	}
	return result
}

func appendUnique(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func safeID(value string) string {
	var builder strings.Builder
	lastWasSeparator := false
	for _, valueRune := range strings.ToLower(value) {
		if (valueRune >= 'a' && valueRune <= 'z') || (valueRune >= '0' && valueRune <= '9') {
			builder.WriteRune(valueRune)
			lastWasSeparator = false
		} else if builder.Len() > 0 && !lastWasSeparator {
			builder.WriteByte('-')
			lastWasSeparator = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func displayUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
