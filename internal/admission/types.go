// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package admission

import (
	"time"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

const RequestSchemaVersion = "breachsafe.report.request.community-single-scan/v1alpha1"

type Request struct {
	SchemaVersion            string                 `json:"schema_version"`
	Identity                 evidence.Identity      `json:"identity"`
	CBOM                     SourceDeclaration      `json:"cbom"`
	ScanJSON                 SourceDeclaration      `json:"scan_json"`
	RenderOptions            evidence.RenderOptions `json:"render_options"`
	AllowCorrelationMismatch bool                   `json:"allow_correlation_mismatch"`
}

type SourceDeclaration struct {
	MediaType      string `json:"media_type"`
	Schema         string `json:"schema"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
}

type Result struct {
	Model               evidence.CommunitySingleScan
	RequestBytesSHA256  string
	CBOMBytesSHA256     string
	ScanJSONBytesSHA256 string
}

type qureddyDocument struct {
	SchemaVersion string              `json:"schema_version"`
	Scan          qureddyScan         `json:"scan"`
	Target        qureddyTarget       `json:"target"`
	Dependencies  []qureddyDependency `json:"dependencies"`
	Assets        []qureddyAsset      `json:"assets"`
	Evidence      []qureddyEvidence   `json:"evidence"`
	Findings      []qureddyFinding    `json:"findings"`
	Summary       qureddySummary      `json:"summary"`
}

type qureddyScan struct {
	ScanID         string    `json:"scan_id"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
	ScannerName    string    `json:"scanner_name"`
	ScannerVersion string    `json:"scanner_version"`
	Status         string    `json:"status"`
	TotalAttempts  int       `json:"total_attempts"`
}

type qureddyTarget struct {
	OriginalInput string  `json:"original_input"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	SNI           *string `json:"sni"`
	Scheme        string  `json:"scheme"`
	Locator       string  `json:"locator"`
}

type qureddyDependency struct {
	Name                   string  `json:"name"`
	Version                *string `json:"version"`
	SupportsTLS13Groups    bool    `json:"supports_tls13_groups"`
	SupportsX25519MLKEM768 bool    `json:"supports_x25519mlkem768"`
	FailureCategory        *string `json:"failure_category"`
}

type qureddyAsset struct {
	ID                     string  `json:"id"`
	AssetType              string  `json:"asset_type"`
	Locator                string  `json:"locator"`
	DisplayName            string  `json:"display_name"`
	Protocol               string  `json:"protocol"`
	ProtocolVersion        *string `json:"protocol_version"`
	Algorithm              *string `json:"algorithm"`
	Primitive              *string `json:"primitive"`
	ParameterSetIdentifier *string `json:"parameter_set_identifier"`
	KeySize                *int    `json:"key_size"`
	NegotiatedGroup        *string `json:"negotiated_group"`
	BomRef                 *string `json:"bom_ref"`
	OID                    *string `json:"oid"`
	NISTQuantumLevel       *int    `json:"nist_quantum_security_level"`
}

type qureddyEvidence struct {
	ID              string              `json:"id"`
	AssetID         string              `json:"asset_id"`
	EvidenceType    string              `json:"evidence_type"`
	ObservationType string              `json:"observation_type"`
	Source          string              `json:"source"`
	Protocol        string              `json:"protocol"`
	ProtocolVersion *string             `json:"protocol_version"`
	CipherSuite     *string             `json:"cipher_suite"`
	NegotiatedGroup *string             `json:"negotiated_group"`
	ProbeRole       *string             `json:"probe_role"`
	ExpectedGroup   *string             `json:"expected_group"`
	ProbeResult     *qureddyProbeResult `json:"probe_result"`
	FailureCategory *string             `json:"failure_category"`
	Confidence      string              `json:"confidence"`
	Notes           []string            `json:"notes"`
}

type qureddyProbeResult struct {
	ReturnCode      int     `json:"return_code"`
	StdoutSHA256    string  `json:"stdout_sha256"`
	StderrSHA256    string  `json:"stderr_sha256"`
	DurationMS      int64   `json:"duration_ms"`
	AttemptNumber   int     `json:"attempt_number"`
	FailureCategory *string `json:"failure_category"`
}

type qureddyFinding struct {
	ID                     string   `json:"id"`
	AssetID                string   `json:"asset_id"`
	EvidenceIDs            []string `json:"evidence_ids"`
	RuleID                 string   `json:"rule_id"`
	FindingType            string   `json:"finding_type"`
	Title                  string   `json:"title"`
	Description            string   `json:"description"`
	Severity               string   `json:"severity"`
	Readiness              string   `json:"readiness"`
	Confidence             string   `json:"confidence"`
	Algorithm              *string  `json:"algorithm"`
	Primitive              *string  `json:"primitive"`
	ParameterSetIdentifier *string  `json:"parameter_set_identifier"`
	KeySize                *int     `json:"key_size"`
	Protocol               string   `json:"protocol"`
	ProtocolVersion        *string  `json:"protocol_version"`
	NegotiatedGroup        *string  `json:"negotiated_group"`
	BomRef                 *string  `json:"bom_ref"`
	OID                    *string  `json:"oid"`
	NISTQuantumLevel       *int     `json:"nist_quantum_security_level"`
}

type qureddySummary struct {
	Target          string  `json:"target"`
	FindingCount    int     `json:"finding_count"`
	HighestSeverity *string `json:"highest_severity"`
	Readiness       string  `json:"readiness"`
	FailureCategory *string `json:"failure_category"`
}

type cycloneDXDocument struct {
	Schema       string                `json:"$schema"`
	BomFormat    string                `json:"bomFormat"`
	SpecVersion  string                `json:"specVersion"`
	SerialNumber string                `json:"serialNumber"`
	Version      int                   `json:"version"`
	Metadata     cycloneDXMetadata     `json:"metadata"`
	Components   []cycloneDXComponent  `json:"components"`
	Dependencies []cycloneDXDependency `json:"dependencies"`
	Annotations  []cycloneDXAnnotation `json:"annotations"`
}

type cycloneDXMetadata struct {
	Timestamp  time.Time           `json:"timestamp"`
	Component  cycloneDXComponent  `json:"component"`
	Tools      cycloneDXTools      `json:"tools"`
	Properties []cycloneDXProperty `json:"properties"`
}

type cycloneDXTools struct {
	Components []cycloneDXComponent `json:"components"`
}

type cycloneDXComponent struct {
	BomRef           string                     `json:"bom-ref"`
	Type             string                     `json:"type"`
	Name             string                     `json:"name"`
	Version          string                     `json:"version,omitempty"`
	CryptoProperties *cycloneDXCryptoProperties `json:"cryptoProperties,omitempty"`
	Properties       []cycloneDXProperty        `json:"properties,omitempty"`
	Evidence         *cycloneDXEvidence         `json:"evidence,omitempty"`
}

type cycloneDXCryptoProperties struct {
	AssetType             string                          `json:"assetType"`
	AlgorithmProperties   *cycloneDXAlgorithmProperties   `json:"algorithmProperties,omitempty"`
	ProtocolProperties    *cycloneDXProtocolProperties    `json:"protocolProperties,omitempty"`
	CertificateProperties *cycloneDXCertificateProperties `json:"certificateProperties,omitempty"`
}

type cycloneDXAlgorithmProperties struct {
	Primitive                string   `json:"primitive,omitempty"`
	ParameterSetIdentifier   string   `json:"parameterSetIdentifier,omitempty"`
	Curve                    string   `json:"curve,omitempty"`
	CryptoFunctions          []string `json:"cryptoFunctions,omitempty"`
	ClassicalSecurityLevel   *int     `json:"classicalSecurityLevel,omitempty"`
	NISTQuantumSecurityLevel *int     `json:"nistQuantumSecurityLevel,omitempty"`
}

type cycloneDXProtocolProperties struct {
	Type         string                 `json:"type,omitempty"`
	Version      string                 `json:"version,omitempty"`
	CipherSuites []cycloneDXCipherSuite `json:"cipherSuites,omitempty"`
}

type cycloneDXCipherSuite struct {
	Name       string   `json:"name"`
	Algorithms []string `json:"algorithms"`
}

type cycloneDXCertificateProperties struct {
	CertificateFormat     string `json:"certificateFormat,omitempty"`
	IssuerName            string `json:"issuerName,omitempty"`
	SubjectName           string `json:"subjectName,omitempty"`
	NotValidBefore        string `json:"notValidBefore,omitempty"`
	NotValidAfter         string `json:"notValidAfter,omitempty"`
	SignatureAlgorithmRef string `json:"signatureAlgorithmRef,omitempty"`
	SubjectPublicKeyRef   string `json:"subjectPublicKeyRef,omitempty"`
	SerialNumber          string `json:"serialNumber,omitempty"`
}

type cycloneDXProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type cycloneDXEvidence struct {
	Occurrences []cycloneDXOccurrence `json:"occurrences"`
}

type cycloneDXOccurrence struct {
	Location          string `json:"location"`
	AdditionalContext string `json:"additionalContext"`
}

type cycloneDXDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn,omitempty"`
	Provides  []string `json:"provides,omitempty"`
}

type cycloneDXAnnotation struct {
	BomRef    string    `json:"bom-ref"`
	Subjects  []string  `json:"subjects"`
	Timestamp time.Time `json:"timestamp"`
	Text      string    `json:"text"`
}
