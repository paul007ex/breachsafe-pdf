// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package admission

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
)

// Limits bounds what admission will read and decode. Every limit is enforced before
// the bytes are parsed, so a hostile artifact cannot exhaust memory during decode.
type Limits struct {
	MaxRequestBytes  int
	MaxCBOMBytes     int
	MaxScanJSONBytes int
	MaxComponents    int
	MaxAnnotations   int
	MaxAssets        int
	MaxEvidence      int
	MaxFindings      int
	Model            evidence.Limits
}

// DefaultLimits returns the limits admission applies when a caller supplies none.
func DefaultLimits() Limits {
	return Limits{
		MaxRequestBytes:  64 << 10,
		MaxCBOMBytes:     10 << 20,
		MaxScanJSONBytes: 10 << 20,
		MaxComponents:    5_000,
		MaxAnnotations:   1_000,
		MaxAssets:        5_000,
		MaxEvidence:      10_000,
		MaxFindings:      1_000,
		Model:            evidence.DefaultLimits(),
	}
}

var scanTopLevelFields = stringSet(
	"schema_version", "scan", "target", "dependencies", "assets", "evidence", "findings", "summary",
)

var cbomTopLevelFields = stringSet(
	"$schema", "bomFormat", "specVersion", "serialNumber", "version", "metadata", "components", "services",
	"externalReferences", "dependencies", "compositions", "vulnerabilities", "annotations", "formulation",
	"declarations", "definitions", "properties", "signature",
)

var readinessValues = stringSet(
	"quantum_vulnerable", "classically_weak", "transitional_hybrid", "quantum_safe", "unknown", "not_applicable",
)

var severityValues = stringSet("critical", "high", "medium", "low", "info")
var confidenceValues = stringSet("high", "medium", "low")
var observationValues = stringSet("negotiated", "offered", "observed", "inferred", "not_offered", "not_testable")
var failureValues = stringSet(
	"local_openssl_missing", "local_openssl_broken", "local_openssl_version_unreadable", "local_openssl_is_libressl",
	"local_openssl_too_old", "local_openssl_version_mismatch", "local_openssl_lacks_group", "target_scan_failed",
	"target_connect_failed", "tls_handshake_failed", "sni_required_or_wrong", "middlebox_or_mtu_failure",
	"parse_no_group", "parse_ambiguous", "unexpected_group",
)

func validateRequest(request Request) error {
	if request.SchemaVersion != RequestSchemaVersion {
		return fault.New(fault.CodeSchemaUnsupported, "admission.request", "schema_version", "unsupported report request schema")
	}
	if request.CBOM.MediaType != "application/vnd.cyclonedx+json" || request.CBOM.Schema != "cyclonedx:1.7" {
		return fault.New(fault.CodeSchemaUnsupported, "admission.request", "cbom", "requires application/vnd.cyclonedx+json and cyclonedx:1.7")
	}
	if request.ScanJSON.MediaType != "application/json" || request.ScanJSON.Schema != "qureddy.scan.v1" {
		return fault.New(fault.CodeSchemaUnsupported, "admission.request", "scan_json", "requires application/json and qureddy.scan.v1")
	}
	for field, value := range map[string]string{
		"cbom.expected_sha256":      request.CBOM.ExpectedSHA256,
		"scan_json.expected_sha256": request.ScanJSON.ExpectedSHA256,
	} {
		if value != "" && !isSHA256(value) {
			return fault.New(fault.CodeInvalidInput, "admission.request", field, "must be 64 lowercase hexadecimal characters")
		}
	}
	return nil
}

func validateQuReddy(ctx context.Context, scan qureddyDocument, limits Limits) error {
	if err := ctx.Err(); err != nil {
		return fault.Wrap(fault.CodeCanceled, "admission.scan_validate", err)
	}
	if scan.SchemaVersion != "qureddy.scan.v1" {
		return fault.New(fault.CodeSchemaUnsupported, "admission.scan_validate", "schema_version", "requires qureddy.scan.v1")
	}
	if scan.Scan.ScanID == "" || scan.Scan.StartedAt.IsZero() || scan.Scan.CompletedAt.IsZero() || scan.Scan.CompletedAt.Before(scan.Scan.StartedAt) {
		return inputInvalid("scan", "scan identity and ordered timestamps are required")
	}
	if !isSupportedScanner(scan.Scan.ScannerName) {
		return inputInvalid("scan.scanner_name", "must be tls, ssh, or ike")
	}
	failedScan := !isNonFailureScanStatus(scan.Scan.ScannerName, scan.Scan.Status)
	if failedScan {
		if _, ok := failureValues[scan.Scan.Status]; !ok {
			return inputInvalid("scan.status", "unsupported failure category")
		}
	}
	if scan.Scan.TotalAttempts < 0 || scan.Target.Port < 1 || scan.Target.Port > 65535 {
		return inputInvalid("scan", "attempt count and target port are out of range")
	}
	if scan.Target.Scheme != scan.Scan.ScannerName || scan.Target.Host == "" || scan.Target.Locator == "" {
		return inputInvalid("target", "scheme, host, and canonical locator must agree with scanner")
	}
	parsed, err := url.Parse(scan.Target.Locator)
	if err != nil || parsed.Scheme != scan.Target.Scheme || parsed.Host == "" {
		return inputInvalid("target.locator", "must use the scanner's canonical locator scheme")
	}
	if scan.Summary.Target != scan.Target.Locator || scan.Summary.FindingCount != len(scan.Findings) || scan.Summary.FindingCount < 0 {
		return inputInvalid("summary", "target and finding count must match the document")
	}
	if _, ok := readinessValues[scan.Summary.Readiness]; !ok {
		return inputInvalid("summary.readiness", "unsupported readiness")
	}
	if scan.Summary.HighestSeverity != nil {
		if _, ok := severityValues[*scan.Summary.HighestSeverity]; !ok {
			return inputInvalid("summary.highest_severity", "unsupported severity")
		}
	}
	if scan.Summary.FailureCategory != nil {
		if _, ok := failureValues[*scan.Summary.FailureCategory]; !ok || scan.Scan.Status != *scan.Summary.FailureCategory {
			return inputInvalid("summary.failure_category", "must be a supported category equal to scan.status")
		}
	} else if failedScan {
		return inputInvalid("summary.failure_category", "required for a failed scan")
	}
	if err := maximum("assets", len(scan.Assets), limits.MaxAssets); err != nil {
		return err
	}
	if err := maximum("evidence", len(scan.Evidence), limits.MaxEvidence); err != nil {
		return err
	}
	if err := maximum("findings", len(scan.Findings), limits.MaxFindings); err != nil {
		return err
	}

	assetIDs := make(map[string]struct{}, len(scan.Assets))
	for index, asset := range scan.Assets {
		if asset.ID == "" || asset.AssetType == "" || asset.Locator != scan.Target.Locator {
			return inputInvalid(fmt.Sprintf("assets[%d]", index), "stable ID, type, and matching locator are required")
		}
		if _, exists := assetIDs[asset.ID]; exists {
			return inputInvalid(fmt.Sprintf("assets[%d].id", index), "duplicate stable ID")
		}
		assetIDs[asset.ID] = struct{}{}
	}
	evidenceIDs := make(map[string]struct{}, len(scan.Evidence))
	for index, item := range scan.Evidence {
		if item.ID == "" || item.EvidenceType == "" || item.Source == "" {
			return inputInvalid(fmt.Sprintf("evidence[%d]", index), "stable ID, type, and source are required")
		}
		if _, exists := evidenceIDs[item.ID]; exists {
			return inputInvalid(fmt.Sprintf("evidence[%d].id", index), "duplicate stable ID")
		}
		evidenceIDs[item.ID] = struct{}{}
		if _, ok := assetIDs[item.AssetID]; !ok {
			return inputInvalid(fmt.Sprintf("evidence[%d].asset_id", index), "dangling asset reference")
		}
		if _, ok := observationValues[item.ObservationType]; !ok {
			return inputInvalid(fmt.Sprintf("evidence[%d].observation_type", index), "unsupported observation type")
		}
		if _, ok := confidenceValues[item.Confidence]; !ok {
			return inputInvalid(fmt.Sprintf("evidence[%d].confidence", index), "unsupported confidence")
		}
		if item.FailureCategory != nil {
			if _, ok := failureValues[*item.FailureCategory]; !ok {
				return inputInvalid(fmt.Sprintf("evidence[%d].failure_category", index), "unsupported failure category")
			}
		}
	}
	findingIDs := make(map[string]struct{}, len(scan.Findings))
	for index, finding := range scan.Findings {
		if finding.ID == "" || finding.RuleID == "" || finding.Title == "" || len(finding.EvidenceIDs) == 0 {
			return inputInvalid(fmt.Sprintf("findings[%d]", index), "stable ID, rule, title, and evidence are required")
		}
		if _, exists := findingIDs[finding.ID]; exists {
			return inputInvalid(fmt.Sprintf("findings[%d].id", index), "duplicate stable ID")
		}
		findingIDs[finding.ID] = struct{}{}
		if _, ok := assetIDs[finding.AssetID]; !ok {
			return inputInvalid(fmt.Sprintf("findings[%d].asset_id", index), "dangling asset reference")
		}
		for _, evidenceID := range finding.EvidenceIDs {
			if _, ok := evidenceIDs[evidenceID]; !ok {
				return inputInvalid(fmt.Sprintf("findings[%d].evidence_ids", index), "dangling evidence reference")
			}
		}
		if _, ok := severityValues[finding.Severity]; !ok {
			return inputInvalid(fmt.Sprintf("findings[%d].severity", index), "unsupported severity")
		}
		if _, ok := readinessValues[finding.Readiness]; !ok {
			return inputInvalid(fmt.Sprintf("findings[%d].readiness", index), "unsupported readiness")
		}
		if _, ok := confidenceValues[finding.Confidence]; !ok {
			return inputInvalid(fmt.Sprintf("findings[%d].confidence", index), "unsupported confidence")
		}
	}
	return nil
}

func validateCycloneDX(ctx context.Context, cbom cycloneDXDocument, limits Limits) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fault.Wrap(fault.CodeCanceled, "admission.cbom_validate", err)
	}
	if cbom.BomFormat != "CycloneDX" || cbom.SpecVersion != "1.7" {
		return nil, fault.New(fault.CodeSchemaUnsupported, "admission.cbom_validate", "specVersion", "requires CycloneDX 1.7")
	}
	if cbom.Schema != "" && cbom.Schema != "http://cyclonedx.org/schema/bom-1.7.schema.json" && cbom.Schema != "https://cyclonedx.org/schema/bom-1.7.schema.json" {
		return nil, fault.New(fault.CodeSchemaMismatch, "admission.cbom_validate", "$schema", "declared schema does not match CycloneDX 1.7")
	}
	if cbom.SerialNumber == "" || cbom.Version < 1 || cbom.Metadata.Timestamp.IsZero() || cbom.Metadata.Component.BomRef == "" {
		return nil, inputInvalid("cbom", "serial number, version, metadata time, and root component are required")
	}
	if err := maximum("components", len(cbom.Components), limits.MaxComponents); err != nil {
		return nil, err
	}
	if err := maximum("annotations", len(cbom.Annotations), limits.MaxAnnotations); err != nil {
		return nil, err
	}
	properties, err := propertyMap(cbom.Metadata.Properties, "metadata.properties")
	if err != nil {
		return nil, err
	}
	for _, required := range []string{"qureddy:scan.scanner_name", "qureddy:scan.status", "qureddy:scan.readiness", "qureddy:scan.finding_count", "qureddy:target.locator"} {
		if strings.TrimSpace(properties[required]) == "" {
			return nil, inputInvalid("metadata.properties", "missing "+required)
		}
	}
	if !isSupportedScanner(properties["qureddy:scan.scanner_name"]) {
		return nil, inputInvalid("metadata.properties", "unsupported scanner name")
	}
	if _, ok := readinessValues[properties["qureddy:scan.readiness"]]; !ok {
		return nil, inputInvalid("metadata.properties", "unsupported readiness")
	}
	if !isNonFailureScanStatus(properties["qureddy:scan.scanner_name"], properties["qureddy:scan.status"]) {
		if _, ok := failureValues[properties["qureddy:scan.status"]]; !ok {
			return nil, inputInvalid("metadata.properties", "unsupported scan status")
		}
	}
	count, err := strconv.Atoi(properties["qureddy:scan.finding_count"])
	if err != nil || count < 0 || count != len(cbom.Annotations) {
		return nil, inputInvalid("metadata.properties", "finding count must equal annotations")
	}

	refs := map[string]struct{}{cbom.Metadata.Component.BomRef: {}}
	for index, tool := range cbom.Metadata.Tools.Components {
		if tool.BomRef == "" || tool.Name == "" {
			return nil, inputInvalid(fmt.Sprintf("metadata.tools.components[%d]", index), "bom-ref and name are required")
		}
		if _, exists := refs[tool.BomRef]; exists {
			return nil, inputInvalid(fmt.Sprintf("metadata.tools.components[%d].bom-ref", index), "duplicate bom-ref")
		}
		refs[tool.BomRef] = struct{}{}
	}
	for index, component := range cbom.Components {
		if component.BomRef == "" || component.Name == "" || component.Type != "cryptographic-asset" || component.CryptoProperties == nil || component.CryptoProperties.AssetType == "" {
			return nil, inputInvalid(fmt.Sprintf("components[%d]", index), "complete cryptographic asset identity is required")
		}
		if _, exists := refs[component.BomRef]; exists {
			return nil, inputInvalid(fmt.Sprintf("components[%d].bom-ref", index), "duplicate bom-ref")
		}
		refs[component.BomRef] = struct{}{}
		if _, err := propertyMap(component.Properties, fmt.Sprintf("components[%d].properties", index)); err != nil {
			return nil, err
		}
	}
	annotationRefs := make(map[string]struct{}, len(cbom.Annotations))
	for index, annotation := range cbom.Annotations {
		if annotation.BomRef == "" || annotation.Timestamp.IsZero() || strings.TrimSpace(annotation.Text) == "" || len(annotation.Subjects) == 0 {
			return nil, inputInvalid(fmt.Sprintf("annotations[%d]", index), "identity, time, text, and subject are required")
		}
		if _, exists := annotationRefs[annotation.BomRef]; exists {
			return nil, inputInvalid(fmt.Sprintf("annotations[%d].bom-ref", index), "duplicate bom-ref")
		}
		annotationRefs[annotation.BomRef] = struct{}{}
		for _, subject := range annotation.Subjects {
			if _, ok := refs[subject]; !ok {
				return nil, inputInvalid(fmt.Sprintf("annotations[%d].subjects", index), "dangling component reference")
			}
		}
	}
	for index, dependency := range cbom.Dependencies {
		if _, ok := refs[dependency.Ref]; !ok {
			return nil, inputInvalid(fmt.Sprintf("dependencies[%d].ref", index), "dangling dependency reference")
		}
		for _, ref := range append(append([]string{}, dependency.DependsOn...), dependency.Provides...) {
			if _, ok := refs[ref]; !ok {
				return nil, inputInvalid(fmt.Sprintf("dependencies[%d]", index), "dangling relationship reference")
			}
		}
	}
	return properties, nil
}

func propertyMap(properties []cycloneDXProperty, field string) (map[string]string, error) {
	result := make(map[string]string, len(properties))
	for _, property := range properties {
		if property.Name == "" {
			return nil, inputInvalid(field, "property name must not be empty")
		}
		if _, exists := result[property.Name]; exists {
			return nil, inputInvalid(field, "duplicate property "+property.Name)
		}
		result[property.Name] = property.Value
	}
	return result, nil
}

func maximum(field string, actual, limit int) error {
	if actual > limit {
		return fault.Format(fault.CodeLimitExceeded, "admission.validate", field, "got %d; maximum is %d", actual, limit)
	}
	return nil
}

func inputInvalid(field, detail string) error {
	return fault.New(fault.CodeInvalidInput, "admission.validate", field, detail)
}

func isSupportedScanner(value string) bool {
	switch value {
	case "tls", "ssh", "ike":
		return true
	default:
		return false
	}
}

func isNonFailureScanStatus(scanner, status string) bool {
	return isCompletedObservationStatus(scanner, status) || (scanner == "ike" && status == "no_response")
}

func isCompletedObservationStatus(scanner, status string) bool {
	return status == "completed" || (scanner == "ike" && status == "rejected")
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, valueRune := range value {
		if (valueRune < '0' || valueRune > '9') && (valueRune < 'a' || valueRune > 'f') {
			return false
		}
	}
	return true
}
