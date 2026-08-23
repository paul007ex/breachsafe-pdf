// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package evidence defines the source-neutral Community Single-Scan report
// contract. It deliberately contains no FPDF, ePack, tenant, entitlement, or
// producer implementation types.
package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/paul007ex/breachsafe-pdf/internal/fault"
)

const (
	// SchemaVersion identifies the Community report model contract.
	SchemaVersion       = "breachsafe.report.community-single-scan/v1alpha1"
	InputContract       = "breachsafe.report.input.qureddy/v1alpha1"
	RenderResultVersion = "breachsafe.report.render-result/v1alpha1"
)

// RunStatus describes the producer run lifecycle state.
type RunStatus string

const (
	// RunCompleted indicates a completed producer run.
	RunCompleted      RunStatus = "completed"
	RunCompletedEmpty RunStatus = "completed_empty"
	RunPartial        RunStatus = "partial"
	RunFailed         RunStatus = "failed"
	RunCanceled       RunStatus = "cancelled"
	RunUnavailable    RunStatus = "unavailable"
	RunUnknown        RunStatus = "unknown"
)

// CoverageStatus describes how much requested evidence was assessed.
type CoverageStatus string

const (
	// CoverageComplete indicates all requested evidence was assessed.
	CoverageComplete    CoverageStatus = "complete"
	CoveragePartial     CoverageStatus = "partial"
	CoverageUnavailable CoverageStatus = "unavailable"
	CoverageUnknown     CoverageStatus = "unknown"
)

// Authority identifies the provenance strength of a report value.
type Authority string

const (
	// AuthorityObserved identifies a directly observed value.
	AuthorityObserved     Authority = "observed"
	AuthorityToolAsserted Authority = "tool_asserted"
	AuthorityImported     Authority = "imported"
	AuthorityInferred     Authority = "inferred"
	AuthorityGoverned     Authority = "governed"
	AuthorityUnknown      Authority = "unknown"
)

// ValidationStatus describes validation of an input or artifact.
type ValidationStatus string

const (
	// ValidationValid indicates successful validation.
	ValidationValid       ValidationStatus = "valid"
	ValidationInvalid     ValidationStatus = "invalid"
	ValidationUnsupported ValidationStatus = "unsupported"
	ValidationNotRun      ValidationStatus = "not_run"
	ValidationQuarantined ValidationStatus = "quarantined"
)

// PageSize selects the PDF page geometry.
type PageSize string

const (
	// PageA4 selects ISO A4 paper.
	PageA4     PageSize = "a4"
	PageLetter PageSize = "letter"
)

// ColorMode selects color rendering behavior.
type ColorMode string

const (
	// ColorFull selects the full-color palette.
	ColorFull      ColorMode = "color"
	ColorGrayscale ColorMode = "grayscale"
)

// Identity identifies a generated report.
type Identity struct {
	ReportID                string    `json:"report_id"`
	Name                    string    `json:"name"`
	GeneratedAt             time.Time `json:"generated_at"`
	Language                string    `json:"language"`
	OrganizationDisplayName string    `json:"organization_display_name,omitempty"`
	Classification          string    `json:"classification"`
}

// Run records producer execution timing and status.
type Run struct {
	ID                   string    `json:"id"`
	CorrelationID        string    `json:"correlation_id,omitempty"`
	Status               RunStatus `json:"status"`
	StartedAt            time.Time `json:"started_at"`
	CompletedAt          time.Time `json:"completed_at"`
	DurationMilliseconds int64     `json:"duration_milliseconds"`
	Attempt              int       `json:"attempt"`
}

// Subject identifies the scanned target.
type Subject struct {
	ID               string   `json:"id"`
	Kind             string   `json:"kind"`
	DisplayName      string   `json:"display_name"`
	CanonicalAddress string   `json:"canonical_address"`
	SourceRefs       []string `json:"source_refs"`
}

// Tool records a producer or supporting tool.
type Tool struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Role             string   `json:"role"`
	BuildCommit      string   `json:"build_commit,omitempty"`
	ExecutableDigest string   `json:"executable_digest,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	State            string   `json:"state"`
	Limitations      []string `json:"limitations,omitempty"`
}

// Digest records a content digest and algorithm.
type Digest struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

// ValidationResult records an artifact validation outcome.
type ValidationResult struct {
	Validator string           `json:"validator"`
	Version   string           `json:"version"`
	Status    ValidationStatus `json:"status"`
	Detail    string           `json:"detail,omitempty"`
}

// Artifact describes an exact evidence input or output.
type Artifact struct {
	ID                    string             `json:"id"`
	Role                  string             `json:"role"`
	DisplayName           string             `json:"display_name"`
	MediaType             string             `json:"media_type"`
	Schema                string             `json:"schema"`
	DeclaredSchemaVersion string             `json:"declared_schema_version"`
	Digest                Digest             `json:"digest"`
	SizeBytes             int                `json:"size_bytes"`
	ProducerToolRef       string             `json:"producer_tool_ref"`
	ProducedAt            time.Time          `json:"produced_at,omitempty"`
	Validations           []ValidationResult `json:"validations"`
	Relationship          string             `json:"relationship"`
}

// Coverage records evidence collection accounting.
type Coverage struct {
	Status      CoverageStatus `json:"status"`
	Requested   int            `json:"requested"`
	Attempted   int            `json:"attempted"`
	Completed   int            `json:"completed"`
	NotAssessed []string       `json:"not_assessed,omitempty"`
	Authority   Authority      `json:"authority"`
	SourceRefs  []string       `json:"source_refs"`
	Derivation  string         `json:"derivation,omitempty"`
}

// PostureAxis records one bounded readiness dimension.
type PostureAxis struct {
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	Value      string    `json:"value"`
	Authority  Authority `json:"authority"`
	Confidence string    `json:"confidence"`
	SourceRefs []string  `json:"source_refs"`
	ObservedAt time.Time `json:"observed_at,omitempty"`
	Limitation string    `json:"limitation,omitempty"`
	Derivation string    `json:"derivation,omitempty"`
}

// EvidenceRef points to supporting bytes in an admitted artifact.
type EvidenceRef struct {
	ArtifactRef string `json:"artifact_ref"`
	Pointer     string `json:"pointer"`
	Description string `json:"description"`
}

// Finding records one producer observation projected into the report.
type Finding struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Outcome      string        `json:"outcome"`
	Severity     string        `json:"severity"`
	Readiness    string        `json:"readiness"`
	Confidence   string        `json:"confidence"`
	Authority    Authority     `json:"authority"`
	SubjectRef   string        `json:"subject_ref"`
	SourceRefs   []string      `json:"source_refs"`
	EvidenceRefs []EvidenceRef `json:"evidence_refs"`
	RuleRef      string        `json:"rule_ref"`
	ObservedAt   time.Time     `json:"observed_at,omitempty"`
	Protocol     string        `json:"protocol,omitempty"`
	Algorithm    string        `json:"algorithm,omitempty"`
}

// InventoryAsset records one cryptographic inventory item.
type InventoryAsset struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	AssetType        string        `json:"asset_type"`
	Primitive        string        `json:"primitive,omitempty"`
	Protocol         string        `json:"protocol,omitempty"`
	ProtocolVersion  string        `json:"protocol_version,omitempty"`
	ParameterSet     string        `json:"parameter_set,omitempty"`
	KeySize          *int          `json:"key_size,omitempty"`
	NISTQuantumLevel *int          `json:"nist_quantum_level,omitempty"`
	Observation      string        `json:"observation"`
	Readiness        string        `json:"readiness"`
	Severity         string        `json:"severity,omitempty"`
	Authority        Authority     `json:"authority"`
	SubjectRef       string        `json:"subject_ref"`
	SourceRefs       []string      `json:"source_refs"`
	EvidenceRefs     []EvidenceRef `json:"evidence_refs"`
}

// FindingCollection records displayed and total finding counts.
type FindingCollection struct {
	TotalCount               int       `json:"total_count"`
	Items                    []Finding `json:"items"`
	OmittedCount             int       `json:"omitted_count"`
	SelectionRule            string    `json:"selection_rule,omitempty"`
	AuthoritativeArtifactRef string    `json:"authoritative_artifact_ref,omitempty"`
}

// InventoryCollection records displayed and total inventory counts.
type InventoryCollection struct {
	TotalCount               int              `json:"total_count"`
	Items                    []InventoryAsset `json:"items"`
	OmittedCount             int              `json:"omitted_count"`
	SelectionRule            string           `json:"selection_rule,omitempty"`
	AuthoritativeArtifactRef string           `json:"authoritative_artifact_ref,omitempty"`
}

// ErrorRecord records a bounded processing error.
type ErrorRecord struct {
	ID           string   `json:"id"`
	Stage        string   `json:"stage"`
	SourceRef    string   `json:"source_ref"`
	Code         string   `json:"code"`
	Description  string   `json:"description"`
	SubjectRef   string   `json:"subject_ref,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

// Limitation records a boundary or incompleteness warning.
type Limitation struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	SourceRefs  []string `json:"source_refs"`
}

// Correlation records the relationship between exact source artifacts.
type Correlation struct {
	State      string   `json:"state"`
	Basis      string   `json:"basis"`
	SourceRefs []string `json:"source_refs"`
}

// RenderOptions controls deterministic PDF presentation.
type RenderOptions struct {
	PageSize             PageSize  `json:"page_size"`
	Timezone             string    `json:"timezone"`
	ColorMode            ColorMode `json:"color_mode"`
	AccessibilityProfile string    `json:"accessibility_profile"`
}

// CommunitySingleScan is the canonical source-neutral report model.
type CommunitySingleScan struct {
	SchemaVersion       string              `json:"schema_version"`
	Identity            Identity            `json:"identity"`
	Run                 Run                 `json:"run"`
	Subject             Subject             `json:"subject"`
	Tools               []Tool              `json:"tools"`
	Artifacts           []Artifact          `json:"artifacts"`
	Correlation         Correlation         `json:"correlation"`
	Coverage            Coverage            `json:"coverage"`
	PostureAxes         []PostureAxis       `json:"posture_axes"`
	Findings            FindingCollection   `json:"findings"`
	Inventory           InventoryCollection `json:"inventory"`
	Errors              []ErrorRecord       `json:"errors,omitempty"`
	Limitations         []Limitation        `json:"limitations"`
	UnresolvedQuestions []string            `json:"unresolved_questions,omitempty"`
	IntegrityStatement  string              `json:"integrity_statement"`
	RenderOptions       RenderOptions       `json:"render_options"`
}

// Limits bounds admission and rendering work.
type Limits struct {
	MaxTools           int
	MaxArtifacts       int
	MaxPostureAxes     int
	MaxFindings        int
	MaxInventory       int
	MaxIssues          int
	MaxIdentifierBytes int
	MaxNameRunes       int
	MaxShortRunes      int
	MaxDetailBytes     int
	MaxTotalTextBytes  int
}

// DefaultLimits returns the conservative report limits.
func DefaultLimits() Limits {
	return Limits{
		MaxTools:           64,
		MaxArtifacts:       100,
		MaxPostureAxes:     32,
		MaxFindings:        1_000,
		MaxInventory:       5_000,
		MaxIssues:          2_000,
		MaxIdentifierBytes: 256,
		MaxNameRunes:       200,
		MaxShortRunes:      256,
		MaxDetailBytes:     16 << 10,
		MaxTotalTextBytes:  10 << 20,
	}
}

// Document is a rendered PDF and its capability metadata.
type Document struct {
	Bytes             []byte
	MediaType         string
	Pages             int
	RendererName      string
	RendererVersion   string
	GeneratorVersion  string
	FontBundleName    string
	FontBundleSHA256  string
	AssetBundleSHA256 string
	Capabilities      []CapabilityResult
}

// CapabilityResult records one renderer capability outcome.
type CapabilityResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Renderer renders an admitted Community report model.
type Renderer interface {
	// Ready lets implementations reject a typed-nil or otherwise unusable
	// receiver without reflection at the orchestration boundary.
	Ready() bool
	Render(context.Context, CommunitySingleScan) (Document, error)
}

// RenderResult records exact output and provenance digests.
type RenderResult struct {
	SchemaVersion          string             `json:"schema_version"`
	ReportID               string             `json:"report_id"`
	ContractVersion        string             `json:"contract_version"`
	InputContract          string             `json:"input_contract"`
	InputProfile           string             `json:"input_profile,omitempty"`
	InputProfileVersion    string             `json:"input_profile_version,omitempty"`
	ReportProfile          string             `json:"report_profile,omitempty"`
	ReportProfileVersion   string             `json:"report_profile_version,omitempty"`
	View                   string             `json:"view"`
	GeneratedAt            time.Time          `json:"generated_at"`
	AdmissionRequestSHA256 string             `json:"admission_request_sha256"`
	ModelSHA256            string             `json:"model_sha256"`
	RenderRequestSHA256    string             `json:"render_request_sha256"`
	PDF                    PDFDescriptor      `json:"pdf"`
	Generator              BuildIdentity      `json:"generator"`
	Renderer               BuildIdentity      `json:"renderer"`
	FontBundle             BundleIdentity     `json:"font_bundle"`
	AssetBundle            BundleIdentity     `json:"asset_bundle"`
	InputArtifacts         []Artifact         `json:"input_artifacts"`
	CapabilityResults      []CapabilityResult `json:"capability_results"`
	Warnings               []string           `json:"warnings"`
}

// PDFDescriptor describes the generated PDF bytes.
type PDFDescriptor struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	SHA256    string `json:"sha256"`
	Bytes     int    `json:"bytes"`
	Pages     int    `json:"pages"`
}

// BuildIdentity identifies a generator or renderer build.
type BuildIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}

// BundleIdentity identifies an embedded asset bundle.
type BundleIdentity struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// Canonicalize returns a deterministic, caller-owned model copy.
func Canonicalize(in CommunitySingleScan) CommunitySingleScan {
	out := in
	out.Tools = slices.Clone(in.Tools)
	out.Artifacts = slices.Clone(in.Artifacts)
	out.PostureAxes = slices.Clone(in.PostureAxes)
	out.Findings.Items = slices.Clone(in.Findings.Items)
	out.Inventory.Items = slices.Clone(in.Inventory.Items)
	out.Errors = slices.Clone(in.Errors)
	out.Limitations = slices.Clone(in.Limitations)
	out.UnresolvedQuestions = slices.Clone(in.UnresolvedQuestions)
	sort.SliceStable(out.Tools, func(i, j int) bool { return out.Tools[i].ID < out.Tools[j].ID })
	sort.SliceStable(out.Artifacts, func(i, j int) bool { return out.Artifacts[i].ID < out.Artifacts[j].ID })
	sort.SliceStable(out.PostureAxes, func(i, j int) bool { return out.PostureAxes[i].ID < out.PostureAxes[j].ID })
	sort.SliceStable(out.Findings.Items, func(i, j int) bool {
		left, right := severityRank(out.Findings.Items[i].Severity), severityRank(out.Findings.Items[j].Severity)
		if left != right {
			return left > right
		}
		return out.Findings.Items[i].ID < out.Findings.Items[j].ID
	})
	sort.SliceStable(out.Inventory.Items, func(i, j int) bool {
		left := out.Inventory.Items[i].AssetType + "\x00" + out.Inventory.Items[i].Name + "\x00" + out.Inventory.Items[i].ID
		right := out.Inventory.Items[j].AssetType + "\x00" + out.Inventory.Items[j].Name + "\x00" + out.Inventory.Items[j].ID
		return left < right
	})
	sort.SliceStable(out.Errors, func(i, j int) bool { return out.Errors[i].ID < out.Errors[j].ID })
	sort.SliceStable(out.Limitations, func(i, j int) bool { return out.Limitations[i].ID < out.Limitations[j].ID })
	sort.Strings(out.UnresolvedQuestions)
	return out
}

// ModelDigest returns the digest of the canonical report model.
func ModelDigest(in CommunitySingleScan) (string, error) {
	data, err := json.Marshal(Canonicalize(in))
	if err != nil {
		return "", fault.Wrap(fault.CodeInvalidInput, "evidence.digest", err)
	}
	return DigestBytes(data), nil
}

// RenderRequestDigest returns the digest of model and render options.
func RenderRequestDigest(in CommunitySingleScan) (string, error) {
	return RenderRequestDigestForView(in, "community_single_scan")
}

// RenderRequestDigestForView includes the selected report view in the digest,
// preventing two projections of the same model from sharing a receipt digest.
func RenderRequestDigestForView(in CommunitySingleScan, view string) (string, error) {
	modelDigest, err := ModelDigest(in)
	if err != nil {
		return "", err
	}
	request := struct {
		Contract    string        `json:"contract"`
		View        string        `json:"view"`
		ModelSHA256 string        `json:"model_sha256"`
		Options     RenderOptions `json:"options"`
	}{SchemaVersion, view, modelDigest, in.RenderOptions}
	data, err := json.Marshal(request)
	if err != nil {
		return "", fault.Wrap(fault.CodeInvalidInput, "evidence.render_request_digest", err)
	}
	return DigestBytes(data), nil
}

// DigestBytes returns the lowercase SHA-256 digest of data.
func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Validate checks the closed report contract and all configured bounds.
func Validate(ctx context.Context, model CommunitySingleScan, limits Limits) error {
	if err := ctx.Err(); err != nil {
		return fault.Wrap(fault.CodeCanceled, "evidence.validate", err)
	}
	if model.SchemaVersion != SchemaVersion {
		return invalid("schema_version", "unsupported Community report schema")
	}
	if !validRunStatus(model.Run.Status) || !validCoverageStatus(model.Coverage.Status) {
		return invalid("state", "unsupported run or coverage state")
	}
	if model.Identity.GeneratedAt.IsZero() || model.Run.StartedAt.IsZero() || model.Run.CompletedAt.IsZero() {
		return invalid("time", "generated, start, and completion times are required")
	}
	if model.Run.CompletedAt.Before(model.Run.StartedAt) || model.Run.DurationMilliseconds < 0 || model.Run.Attempt < 1 {
		return invalid("run", "completion must not precede start; duration must be non-negative; attempt must be positive")
	}
	if model.RenderOptions.PageSize != PageA4 && model.RenderOptions.PageSize != PageLetter {
		return invalid("render_options.page_size", "must be a4 or letter")
	}
	if model.RenderOptions.ColorMode != ColorFull && model.RenderOptions.ColorMode != ColorGrayscale {
		return invalid("render_options.color_mode", "must be color or grayscale")
	}
	if model.RenderOptions.Timezone != "UTC" || model.RenderOptions.AccessibilityProfile != "none" {
		return invalid("render_options", "prototype supports UTC and accessibility_profile=none only")
	}
	for field, value := range map[string]string{
		"identity.report_id": model.Identity.ReportID,
		"run.id":             model.Run.ID,
		"subject.id":         model.Subject.ID,
	} {
		if err := validateIdentifier(field, value, limits); err != nil {
			return err
		}
	}
	for field, value := range map[string]string{
		"identity.name":             model.Identity.Name,
		"identity.language":         model.Identity.Language,
		"identity.classification":   model.Identity.Classification,
		"subject.kind":              model.Subject.Kind,
		"subject.display_name":      model.Subject.DisplayName,
		"subject.canonical_address": model.Subject.CanonicalAddress,
		"correlation.state":         model.Correlation.State,
	} {
		if err := validateShort(field, value, limits, false); err != nil {
			return err
		}
	}
	if err := validateShort("identity.organization_display_name", model.Identity.OrganizationDisplayName, limits, true); err != nil {
		return err
	}
	if err := validateDetail("correlation.basis", model.Correlation.Basis, limits, false); err != nil {
		return err
	}
	if err := validateDetail("integrity_statement", model.IntegrityStatement, limits, false); err != nil {
		return err
	}
	if model.Identity.Language != "en-US" {
		return invalid("identity.language", "prototype supports en-US only")
	}
	if model.Correlation.State != "matched" && model.Correlation.State != "mismatched" && model.Correlation.State != "unknown" {
		return invalid("correlation.state", "must be matched, mismatched, or unknown")
	}
	if err := bounded("tools", len(model.Tools), limits.MaxTools); err != nil {
		return err
	}
	if err := bounded("artifacts", len(model.Artifacts), limits.MaxArtifacts); err != nil {
		return err
	}
	if err := bounded("posture_axes", len(model.PostureAxes), limits.MaxPostureAxes); err != nil {
		return err
	}
	if err := bounded("findings", model.Findings.TotalCount, limits.MaxFindings); err != nil {
		return err
	}
	if err := bounded("inventory", model.Inventory.TotalCount, limits.MaxInventory); err != nil {
		return err
	}
	if err := bounded("issues", len(model.Errors)+len(model.Limitations)+len(model.UnresolvedQuestions), limits.MaxIssues); err != nil {
		return err
	}
	if len(model.Tools) == 0 || len(model.Artifacts) < 2 || len(model.PostureAxes) == 0 {
		return invalid("model", "tools, both source artifacts, and posture axes are required")
	}

	toolIDs := make(map[string]struct{}, len(model.Tools))
	for index, tool := range model.Tools {
		prefix := fmt.Sprintf("tools[%d]", index)
		if err := validateUniqueID(prefix+".id", tool.ID, toolIDs, limits); err != nil {
			return err
		}
		for field, value := range map[string]string{"name": tool.Name, "version": tool.Version, "role": tool.Role, "state": tool.State} {
			if err := validateShort(prefix+"."+field, value, limits, false); err != nil {
				return err
			}
		}
		if !validToolState(tool.State) {
			return invalid(prefix+".state", "unsupported tool state")
		}
		for capabilityIndex, capability := range tool.Capabilities {
			if err := validateShort(fmt.Sprintf("%s.capabilities[%d]", prefix, capabilityIndex), capability, limits, false); err != nil {
				return err
			}
		}
		for limitationIndex, limitation := range tool.Limitations {
			if err := validateDetail(fmt.Sprintf("%s.limitations[%d]", prefix, limitationIndex), limitation, limits, false); err != nil {
				return err
			}
		}
	}
	artifactIDs := make(map[string]struct{}, len(model.Artifacts))
	for index, artifact := range model.Artifacts {
		prefix := fmt.Sprintf("artifacts[%d]", index)
		if err := validateUniqueID(prefix+".id", artifact.ID, artifactIDs, limits); err != nil {
			return err
		}
		if _, ok := toolIDs[artifact.ProducerToolRef]; !ok {
			return invalid(prefix+".producer_tool_ref", "unknown tool reference")
		}
		for field, value := range map[string]string{
			"role": artifact.Role, "display_name": artifact.DisplayName, "media_type": artifact.MediaType,
			"schema": artifact.Schema, "declared_schema_version": artifact.DeclaredSchemaVersion, "relationship": artifact.Relationship,
		} {
			if err := validateShort(prefix+"."+field, value, limits, false); err != nil {
				return err
			}
		}
		if artifact.SizeBytes < 1 || artifact.Digest.Algorithm != "sha256" || !validSHA256(artifact.Digest.Value) {
			return invalid(prefix+".digest", "requires positive size and lowercase SHA-256")
		}
		if len(artifact.Validations) == 0 {
			return invalid(prefix+".validations", "at least one validation is required")
		}
		for validationIndex, validation := range artifact.Validations {
			if !validValidation(validation.Status) {
				return invalid(prefix+".validations", "unsupported validation state")
			}
			for field, value := range map[string]string{"validator": validation.Validator, "version": validation.Version} {
				if err := validateShort(fmt.Sprintf("%s.validations[%d].%s", prefix, validationIndex, field), value, limits, false); err != nil {
					return err
				}
			}
			if err := validateDetail(fmt.Sprintf("%s.validations[%d].detail", prefix, validationIndex), validation.Detail, limits, true); err != nil {
				return err
			}
		}
	}
	if err := validateSourceRefs("correlation.source_refs", model.Correlation.SourceRefs, artifactIDs); err != nil {
		return err
	}
	if err := validateSourceRefs("subject.source_refs", model.Subject.SourceRefs, artifactIDs); err != nil {
		return err
	}
	if err := validateSourceRefs("coverage.source_refs", model.Coverage.SourceRefs, artifactIDs); err != nil {
		return err
	}
	if model.Coverage.Requested < 0 || model.Coverage.Attempted < 0 || model.Coverage.Completed < 0 || model.Coverage.Completed > model.Coverage.Attempted || model.Coverage.Attempted > model.Coverage.Requested {
		return invalid("coverage", "counts must be non-negative and completed <= attempted <= requested")
	}
	if !validAuthority(model.Coverage.Authority) {
		return invalid("coverage.authority", "unsupported authority")
	}
	if err := validateDetail("coverage.derivation", model.Coverage.Derivation, limits, true); err != nil {
		return err
	}
	for index, value := range model.Coverage.NotAssessed {
		if err := validateDetail(fmt.Sprintf("coverage.not_assessed[%d]", index), value, limits, false); err != nil {
			return err
		}
	}

	axisIDs := make(map[string]struct{}, len(model.PostureAxes))
	for index, axis := range model.PostureAxes {
		prefix := fmt.Sprintf("posture_axes[%d]", index)
		if err := validateUniqueID(prefix+".id", axis.ID, axisIDs, limits); err != nil {
			return err
		}
		if err := validateShort(prefix+".label", axis.Label, limits, false); err != nil {
			return err
		}
		// A posture value may contain a complete observed algorithm/cipher
		// inventory from the producer. Keep it bounded as report detail rather
		// than applying the identifier-sized short-string limit.
		if err := validateDetail(prefix+".value", axis.Value, limits, false); err != nil {
			return err
		}
		if !validAuthority(axis.Authority) {
			return invalid(prefix+".authority", "unsupported authority")
		}
		if !validConfidence(axis.Confidence) {
			return invalid(prefix+".confidence", "must be high, medium, low, or unknown")
		}
		if err := validateDetail(prefix+".limitation", axis.Limitation, limits, true); err != nil {
			return err
		}
		if err := validateDetail(prefix+".derivation", axis.Derivation, limits, true); err != nil {
			return err
		}
		if err := validateSourceRefs(prefix+".source_refs", axis.SourceRefs, artifactIDs); err != nil {
			return err
		}
	}
	if err := validateCollection("findings", model.Findings.TotalCount, len(model.Findings.Items), model.Findings.OmittedCount, model.Findings.SelectionRule, model.Findings.AuthoritativeArtifactRef, artifactIDs); err != nil {
		return err
	}
	if err := validateCollection("inventory", model.Inventory.TotalCount, len(model.Inventory.Items), model.Inventory.OmittedCount, model.Inventory.SelectionRule, model.Inventory.AuthoritativeArtifactRef, artifactIDs); err != nil {
		return err
	}
	findingIDs := make(map[string]struct{}, len(model.Findings.Items))
	for index, finding := range model.Findings.Items {
		prefix := fmt.Sprintf("findings.items[%d]", index)
		if err := validateUniqueID(prefix+".id", finding.ID, findingIDs, limits); err != nil {
			return err
		}
		if finding.SubjectRef != model.Subject.ID {
			return invalid(prefix+".subject_ref", "unknown subject reference")
		}
		if !validAuthority(finding.Authority) {
			return invalid(prefix+".authority", "unsupported authority")
		}
		if !validFindingOutcome(finding.Outcome) || !validSeverity(finding.Severity) || !validReadiness(finding.Readiness) || !validConfidence(finding.Confidence) {
			return invalid(prefix, "unsupported outcome, severity, readiness, or confidence")
		}
		if err := validateSourceRefs(prefix+".source_refs", finding.SourceRefs, artifactIDs); err != nil {
			return err
		}
		if err := validateShort(prefix+".title", finding.Title, limits, false); err != nil {
			return err
		}
		if err := validateDetail(prefix+".description", finding.Description, limits, false); err != nil {
			return err
		}
		for field, value := range map[string]string{"rule_ref": finding.RuleRef, "protocol": finding.Protocol, "algorithm": finding.Algorithm} {
			if err := validateShort(prefix+"."+field, value, limits, true); err != nil {
				return err
			}
		}
		for refIndex, ref := range finding.EvidenceRefs {
			if _, ok := artifactIDs[ref.ArtifactRef]; !ok {
				return invalid(prefix+".evidence_refs", "unknown artifact reference")
			}
			if err := validateShort(fmt.Sprintf("%s.evidence_refs[%d].pointer", prefix, refIndex), ref.Pointer, limits, false); err != nil {
				return err
			}
			if err := validateDetail(fmt.Sprintf("%s.evidence_refs[%d].description", prefix, refIndex), ref.Description, limits, false); err != nil {
				return err
			}
		}
	}
	inventoryIDs := make(map[string]struct{}, len(model.Inventory.Items))
	for index, asset := range model.Inventory.Items {
		prefix := fmt.Sprintf("inventory.items[%d]", index)
		if err := validateUniqueID(prefix+".id", asset.ID, inventoryIDs, limits); err != nil {
			return err
		}
		if asset.SubjectRef != model.Subject.ID {
			return invalid(prefix+".subject_ref", "unknown subject reference")
		}
		if !validAuthority(asset.Authority) {
			return invalid(prefix+".authority", "unsupported authority")
		}
		for field, value := range map[string]string{
			"name": asset.Name, "asset_type": asset.AssetType, "primitive": asset.Primitive, "protocol": asset.Protocol,
			"protocol_version": asset.ProtocolVersion, "parameter_set": asset.ParameterSet,
		} {
			if err := validateShort(prefix+"."+field, value, limits, field != "name" && field != "asset_type"); err != nil {
				return err
			}
		}
		if !validObservation(asset.Observation) || !validReadiness(asset.Readiness) || (asset.Severity != "" && !validSeverity(asset.Severity)) {
			return invalid(prefix, "unsupported observation, readiness, or severity")
		}
		if err := validateSourceRefs(prefix+".source_refs", asset.SourceRefs, artifactIDs); err != nil {
			return err
		}
		for refIndex, ref := range asset.EvidenceRefs {
			if _, ok := artifactIDs[ref.ArtifactRef]; !ok {
				return invalid(prefix+".evidence_refs", "unknown artifact reference")
			}
			if err := validateShort(fmt.Sprintf("%s.evidence_refs[%d].pointer", prefix, refIndex), ref.Pointer, limits, false); err != nil {
				return err
			}
			if err := validateDetail(fmt.Sprintf("%s.evidence_refs[%d].description", prefix, refIndex), ref.Description, limits, false); err != nil {
				return err
			}
		}
	}
	errorIDs := make(map[string]struct{}, len(model.Errors))
	for index, issue := range model.Errors {
		prefix := fmt.Sprintf("errors[%d]", index)
		if err := validateUniqueID(prefix+".id", issue.ID, errorIDs, limits); err != nil {
			return err
		}
		if _, ok := artifactIDs[issue.SourceRef]; !ok {
			return invalid(prefix+".source_ref", "unknown artifact reference")
		}
		for field, value := range map[string]string{"stage": issue.Stage, "code": issue.Code, "subject_ref": issue.SubjectRef} {
			if err := validateShort(prefix+"."+field, value, limits, field == "subject_ref"); err != nil {
				return err
			}
		}
		if issue.SubjectRef != "" && issue.SubjectRef != model.Subject.ID {
			return invalid(prefix+".subject_ref", "unknown subject reference")
		}
		if err := validateDetail(prefix+".description", issue.Description, limits, false); err != nil {
			return err
		}
	}
	limitationIDs := make(map[string]struct{}, len(model.Limitations))
	for index, issue := range model.Limitations {
		if err := validateUniqueID(fmt.Sprintf("limitations[%d].id", index), issue.ID, limitationIDs, limits); err != nil {
			return err
		}
		if !validSeverity(issue.Severity) {
			return invalid(fmt.Sprintf("limitations[%d].severity", index), "unsupported severity")
		}
		if err := validateDetail(fmt.Sprintf("limitations[%d].description", index), issue.Description, limits, false); err != nil {
			return err
		}
		if err := validateSourceRefs(fmt.Sprintf("limitations[%d].source_refs", index), issue.SourceRefs, artifactIDs); err != nil {
			return err
		}
	}
	for index, question := range model.UnresolvedQuestions {
		if err := validateDetail(fmt.Sprintf("unresolved_questions[%d]", index), question, limits, false); err != nil {
			return err
		}
	}
	data, err := json.Marshal(model)
	if err != nil {
		return fault.Wrap(fault.CodeInvalidInput, "evidence.validate", err)
	}
	if len(data) > limits.MaxTotalTextBytes {
		return fault.Format(fault.CodeLimitExceeded, "evidence.validate", "model_bytes", "got %d; maximum is %d", len(data), limits.MaxTotalTextBytes)
	}
	return nil
}

func validateCollection(field string, total, displayed, omitted int, rule, artifact string, artifacts map[string]struct{}) error {
	if total < 0 || displayed < 0 || omitted < 0 || total != displayed+omitted {
		return invalid(field, "total must equal displayed plus omitted")
	}
	if omitted > 0 {
		if strings.TrimSpace(rule) == "" {
			return invalid(field+".selection_rule", "required when rows are omitted")
		}
		if _, ok := artifacts[artifact]; !ok {
			return invalid(field+".authoritative_artifact_ref", "required artifact reference is unresolved")
		}
	}
	return nil
}

func validateSourceRefs(field string, refs []string, artifacts map[string]struct{}) error {
	if len(refs) == 0 {
		return invalid(field, "at least one source reference is required")
	}
	for _, ref := range refs {
		if _, ok := artifacts[ref]; !ok {
			return invalid(field, "unknown artifact reference")
		}
	}
	return nil
}

func validateUniqueID(field, value string, seen map[string]struct{}, limits Limits) error {
	if err := validateIdentifier(field, value, limits); err != nil {
		return err
	}
	if _, exists := seen[value]; exists {
		return invalid(field, "duplicate stable identifier")
	}
	seen[value] = struct{}{}
	return nil
}

func validateIdentifier(field, value string, limits Limits) error {
	if strings.TrimSpace(value) == "" || len(value) > limits.MaxIdentifierBytes {
		return invalid(field, "identifier is empty or too long")
	}
	return validateText(field, value, limits.MaxIdentifierBytes, false, false)
}

func validateShort(field, value string, limits Limits, allowEmpty bool) error {
	return validateText(field, value, limits.MaxShortRunes, allowEmpty, true)
}

func validateDetail(field, value string, limits Limits, allowEmpty bool) error {
	if len(value) > limits.MaxDetailBytes {
		return fault.Format(fault.CodeLimitExceeded, "evidence.validate", field, "got %d bytes; maximum is %d", len(value), limits.MaxDetailBytes)
	}
	return validateText(field, value, limits.MaxDetailBytes, allowEmpty, false)
}

func validateText(field, value string, maximum int, allowEmpty, runeLimit bool) error {
	if !utf8.ValidString(value) {
		return invalid(field, "must be valid UTF-8")
	}
	if !allowEmpty && strings.TrimSpace(value) == "" {
		return invalid(field, "must not be empty")
	}
	length := len(value)
	if runeLimit {
		length = utf8.RuneCountInString(value)
	}
	if length > maximum {
		return fault.Format(fault.CodeLimitExceeded, "evidence.validate", field, "got %d; maximum is %d", length, maximum)
	}
	for _, valueRune := range value {
		if valueRune == '\n' || valueRune == '\t' {
			continue
		}
		if unicode.IsControl(valueRune) || (valueRune >= '\u202A' && valueRune <= '\u202E') || (valueRune >= '\u2066' && valueRune <= '\u2069') {
			return invalid(field, "contains a forbidden control character")
		}
	}
	return nil
}

func validRunStatus(value RunStatus) bool {
	switch value {
	case RunCompleted, RunCompletedEmpty, RunPartial, RunFailed, RunCanceled, RunUnavailable, RunUnknown:
		return true
	}
	return false
}

func validCoverageStatus(value CoverageStatus) bool {
	switch value {
	case CoverageComplete, CoveragePartial, CoverageUnavailable, CoverageUnknown:
		return true
	}
	return false
}

func validAuthority(value Authority) bool {
	switch value {
	case AuthorityObserved, AuthorityToolAsserted, AuthorityImported, AuthorityInferred, AuthorityGoverned, AuthorityUnknown:
		return true
	}
	return false
}

func validValidation(value ValidationStatus) bool {
	switch value {
	case ValidationValid, ValidationInvalid, ValidationUnsupported, ValidationNotRun, ValidationQuarantined:
		return true
	}
	return false
}

func validToolState(value string) bool {
	switch value {
	case "available", "completed", "failed", "unavailable", "unknown":
		return true
	}
	return false
}

func validFindingOutcome(value string) bool {
	switch value {
	case "pass", "fail", "manual", "muted", "unknown", "not_applicable":
		return true
	}
	return false
}

func validSeverity(value string) bool {
	switch value {
	case "critical", "high", "medium", "low", "info":
		return true
	}
	return false
}

func validReadiness(value string) bool {
	switch value {
	case "quantum_vulnerable", "classically_weak", "transitional_hybrid", "quantum_safe", "unknown", "not_applicable":
		return true
	}
	return false
}

func validConfidence(value string) bool {
	switch value {
	case "high", "medium", "low", "unknown":
		return true
	}
	return false
}

func validObservation(value string) bool {
	switch value {
	case "negotiated", "offered", "observed", "inferred", "not_offered", "not_testable", "unknown":
		return true
	}
	return false
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func severityRank(value string) int {
	switch strings.ToLower(value) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	}
	return 0
}

func bounded(field string, actual, maximum int) error {
	if actual > maximum {
		return fault.Format(fault.CodeLimitExceeded, "evidence.validate", field, "got %d; maximum is %d", actual, maximum)
	}
	return nil
}

func invalid(field, detail string) error {
	return fault.New(fault.CodeInvalidInput, "evidence.validate", field, detail)
}
