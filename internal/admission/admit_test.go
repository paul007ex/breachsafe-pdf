// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package admission

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
)

func TestAdmitCorrelatesExactProducerBytes(t *testing.T) {
	cbom, scan := sourceBytes(t)
	request := requestBytes(t, "", "", false)

	result, err := Admit(context.Background(), request, cbom, scan, DefaultLimits())
	if err != nil {
		t.Fatalf("Admit() error = %v", err)
	}
	if result.Model.Correlation.State != "matched" {
		t.Fatalf("correlation = %q, want matched", result.Model.Correlation.State)
	}
	if result.CBOMBytesSHA256 != evidence.DigestBytes(cbom) || result.ScanJSONBytesSHA256 != evidence.DigestBytes(scan) {
		t.Fatal("admission did not report exact-byte input digests")
	}
	if result.Model.Findings.TotalCount != 2 || len(result.Model.Findings.Items) != 2 {
		t.Fatalf("findings = %+v", result.Model.Findings)
	}
	if result.Model.Inventory.TotalCount != 4 || len(result.Model.Inventory.Items) != 4 {
		t.Fatalf("inventory = %+v", result.Model.Inventory)
	}
	modelJSON, err := json.Marshal(result.Model)
	if err != nil {
		t.Fatal(err)
	}
	for _, secretAdjacent := range []string{"/private/producer/path", "s_client", "private output intentionally excluded"} {
		if strings.Contains(string(modelJSON), secretAdjacent) {
			t.Fatalf("normalized model leaked excluded producer detail %q", secretAdjacent)
		}
	}
}

func TestAdmitAcceptsMatchingExpectedDigests(t *testing.T) {
	cbom, scan := sourceBytes(t)
	request := requestBytes(t, evidence.DigestBytes(cbom), evidence.DigestBytes(scan), false)
	if _, err := Admit(context.Background(), request, cbom, scan, DefaultLimits()); err != nil {
		t.Fatalf("Admit() error = %v", err)
	}
}

func TestAdmitRejectsDigestMismatch(t *testing.T) {
	cbom, scan := sourceBytes(t)
	request := requestBytes(t, strings.Repeat("0", 64), evidence.DigestBytes(scan), false)
	_, err := Admit(context.Background(), request, cbom, scan, DefaultLimits())
	assertCode(t, err, fault.CodeDigestMismatch)
}

func TestAdmitRejectsNonASCIIHexDigest(t *testing.T) {
	cbom, scan := sourceBytes(t)
	// Thirty-two Arabic-Indic digits occupy exactly 64 UTF-8 bytes, so this
	// catches byte-length checks that accidentally accept Unicode digits.
	request := requestBytes(t, strings.Repeat("١", 32), evidence.DigestBytes(scan), false)
	_, err := Admit(context.Background(), request, cbom, scan, DefaultLimits())
	assertCode(t, err, fault.CodeInvalidInput)
}

func TestAdmitRejectsDeclaredSchemaMismatch(t *testing.T) {
	cbom, scan := sourceBytes(t)
	cbom = []byte(strings.Replace(string(cbom), `"specVersion":"1.7"`, `"specVersion":"1.6"`, 1))
	_, err := Admit(context.Background(), requestBytes(t, "", "", false), cbom, scan, DefaultLimits())
	assertCode(t, err, fault.CodeSchemaMismatch)
}

func TestAdmitRejectsNestedDuplicateKey(t *testing.T) {
	cbom, scan := sourceBytes(t)
	scan = []byte(strings.Replace(string(scan), `"scan_id":"scan-1"`, `"scan_id":"scan-1","scan_id":"shadow"`, 1))
	_, err := Admit(context.Background(), requestBytes(t, "", "", false), cbom, scan, DefaultLimits())
	assertCode(t, err, fault.CodeInvalidInput)
}

func TestAdmitRejectsDanglingEvidenceReference(t *testing.T) {
	cbom, scan := sourceBytes(t)
	scan = []byte(strings.Replace(string(scan), `"evidence_ids":["ev-hybrid"]`, `"evidence_ids":["missing"]`, 1))
	_, err := Admit(context.Background(), requestBytes(t, "", "", false), cbom, scan, DefaultLimits())
	assertCode(t, err, fault.CodeInvalidInput)
}

func TestAdmitRejectsCorrelationMismatchByDefault(t *testing.T) {
	cbom, scan := sourceBytes(t)
	cbom = []byte(strings.Replace(string(cbom), `"value":"tls://example.com:443"`, `"value":"tls://different.example:443"`, 1))
	_, err := Admit(context.Background(), requestBytes(t, "", "", false), cbom, scan, DefaultLimits())
	assertCode(t, err, fault.CodeCorrelationMismatch)
}

func TestAdmitCanRenderExplicitDiagnosticMismatch(t *testing.T) {
	cbom, scan := sourceBytes(t)
	cbom = []byte(strings.Replace(string(cbom), `"value":"tls://example.com:443"`, `"value":"tls://different.example:443"`, 1))
	result, err := Admit(context.Background(), requestBytes(t, "", "", true), cbom, scan, DefaultLimits())
	if err != nil {
		t.Fatalf("Admit() error = %v", err)
	}
	if result.Model.Correlation.State != "mismatched" {
		t.Fatalf("correlation = %q", result.Model.Correlation.State)
	}
}

func TestAdmitPreservesUnknownCorrelationWhenLinkIsAbsent(t *testing.T) {
	cbomBytes, scan := sourceBytes(t)
	var cbom cycloneDXDocument
	if err := json.Unmarshal(cbomBytes, &cbom); err != nil {
		t.Fatal(err)
	}
	properties := cbom.Metadata.Properties[:0]
	for _, property := range cbom.Metadata.Properties {
		if property.Name != "qureddy:scan.id" {
			properties = append(properties, property)
		}
	}
	cbom.Metadata.Properties = properties
	cbomBytes, err := json.Marshal(cbom)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Admit(context.Background(), requestBytes(t, "", "", false), cbomBytes, scan, DefaultLimits())
	if err != nil {
		t.Fatalf("Admit() error = %v", err)
	}
	if result.Model.Correlation.State != "unknown" || !strings.Contains(result.Model.Correlation.Basis, "scan ID") {
		t.Fatalf("correlation = %+v, want absent-link UNKNOWN", result.Model.Correlation)
	}
}

func TestAdmitHonorsCanceledContext(t *testing.T) {
	cbom, scan := sourceBytes(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Admit(ctx, requestBytes(t, "", "", false), cbom, scan, DefaultLimits())
	assertCode(t, err, fault.CodeCanceled)
}

func sourceBytes(t *testing.T) ([]byte, []byte) {
	t.Helper()
	cbom, err := os.ReadFile("testdata/success.cbom.json")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := os.ReadFile("testdata/success.scan.json")
	if err != nil {
		t.Fatal(err)
	}
	return cbom, scan
}

func requestBytes(t *testing.T, cbomDigest, scanDigest string, allowMismatch bool) []byte {
	t.Helper()
	request := Request{
		SchemaVersion: RequestSchemaVersion,
		Identity: evidence.Identity{
			ReportID: "report-test-1", Name: "Quantum Readiness Evidence Report",
			GeneratedAt: time.Date(2026, 4, 26, 0, 5, 0, 0, time.UTC), Language: "en-US",
			OrganizationDisplayName: "Example Organization", Classification: "COMMUNITY / EVIDENCE REPORT",
		},
		CBOM:                     SourceDeclaration{MediaType: "application/vnd.cyclonedx+json", Schema: "cyclonedx:1.7", ExpectedSHA256: cbomDigest},
		ScanJSON:                 SourceDeclaration{MediaType: "application/json", Schema: "qureddy.scan.v1", ExpectedSHA256: scanDigest},
		RenderOptions:            evidence.RenderOptions{PageSize: evidence.PageA4, Timezone: "UTC", ColorMode: evidence.ColorFull, AccessibilityProfile: "none"},
		AllowCorrelationMismatch: allowMismatch,
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertCode(t *testing.T, err error, want fault.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %s", want)
	}
	var typed *fault.Error
	if !errors.As(err, &typed) {
		t.Fatalf("error type = %T, want *fault.Error: %v", err, err)
	}
	if typed.Code != want {
		t.Fatalf("code = %s, want %s: %v", typed.Code, want, err)
	}
}
