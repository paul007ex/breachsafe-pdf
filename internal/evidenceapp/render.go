// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package evidenceapp orchestrates exact-byte admission, rendering, result
// construction, and no-clobber persistence for the Community report.
package evidenceapp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
	"github.com/paul007ex/breachsafe-pdf/internal/input"
	"github.com/paul007ex/breachsafe-pdf/internal/input/adapters/qureddy"
	"github.com/paul007ex/breachsafe-pdf/internal/output"
	"github.com/paul007ex/breachsafe-pdf/internal/report"
	"github.com/paul007ex/breachsafe-pdf/internal/report/profiles/community"
)

const maxPDFBytes = 100 << 20

// FileRequest names the on-disk inputs and outputs for a render, so the caller does not
// have to read them itself.
type FileRequest struct {
	RequestPath  string
	CBOMPath     string
	ScanJSONPath string
	PDFPath      string
	ResultPath   string
}

// Build carries the generator identity stamped into the emitted document.
type Build struct {
	GeneratorVersion string
	GeneratorCommit  string
}

// RenderFiles renders using the default qureddy input adapter and community report
// profile. Use RenderFilesProfile to supply your own.
func RenderFiles(ctx context.Context, request FileRequest, renderer evidence.Renderer, limits admission.Limits, build Build) (evidence.RenderResult, error) {
	return RenderFilesProfile(ctx, request, renderer, limits, build, qureddy.Adapter{}, community.Profile{})
}

// RenderFilesProfile runs the explicit adapter -> report-profile -> renderer
// pipeline. The legacy RenderFiles function remains as a compatibility alias
// for the original community path.
func RenderFilesProfile(ctx context.Context, request FileRequest, renderer evidence.Renderer, limits admission.Limits, build Build, adapter input.Adapter, profile report.Profile) (evidence.RenderResult, error) {
	if adapter == nil || profile == nil {
		return evidence.RenderResult{}, fault.New(fault.CodeInvalidInput, "evidenceapp.render_files", "profile", "input adapter and report profile are required")
	}
	for field, value := range map[string]string{
		"request": request.RequestPath, "cbom": request.CBOMPath, "scan_json": request.ScanJSONPath,
		"pdf": request.PDFPath, "result": request.ResultPath,
	} {
		if strings.TrimSpace(value) == "" {
			return evidence.RenderResult{}, fault.New(fault.CodeInvalidInput, "evidenceapp.render_files", field, "path is required")
		}
	}
	requestBytes, err := readRegularFile(ctx, request.RequestPath, limits.MaxRequestBytes, "request")
	if err != nil {
		return evidence.RenderResult{}, err
	}
	cbomBytes, err := readRegularFile(ctx, request.CBOMPath, limits.MaxCBOMBytes, "cbom")
	if err != nil {
		return evidence.RenderResult{}, err
	}
	scanBytes, err := readRegularFile(ctx, request.ScanJSONPath, limits.MaxScanJSONBytes, "scan_json")
	if err != nil {
		return evidence.RenderResult{}, err
	}
	if adapter.ID() != profile.InputAdapterID() {
		return evidence.RenderResult{}, fault.Format(fault.CodeInvalidInput, "evidenceapp.render_files", "profile", "report profile %q requires input adapter %q, got %q", profile.ID(), profile.InputAdapterID(), adapter.ID())
	}
	admitted, err := adapter.Admit(ctx, input.Input{Request: requestBytes, Artifacts: map[string][]byte{"cbom": cbomBytes, "scan-json": scanBytes}}, limits)
	if err != nil {
		return evidence.RenderResult{}, err
	}
	if err := profile.Validate(ctx, admitted.Result.Model, limits.Model); err != nil {
		return evidence.RenderResult{}, err
	}
	return renderWithProfiles(ctx, admitted.Result, request.PDFPath, request.ResultPath, renderer, limits.Model, build, admitted.Contract, admitted.ID, admitted.Version, profile.ID(), profile.Version(), profile.View())
}

// Render renders an already-admitted result to the given PDF and RenderResult paths.
func Render(ctx context.Context, admitted admission.Result, pdfPath, resultPath string, renderer evidence.Renderer, limits evidence.Limits, build Build) (evidence.RenderResult, error) {
	return renderWithProfiles(ctx, admitted, pdfPath, resultPath, renderer, limits, build, evidence.InputContract, "", "", "", "", "community_single_scan")
}

func renderWithProfiles(ctx context.Context, admitted admission.Result, pdfPath, resultPath string, renderer evidence.Renderer, limits evidence.Limits, build Build, inputContract, inputProfile, inputProfileVersion, reportProfile, reportProfileVersion, view string) (evidence.RenderResult, error) {
	if renderer == nil || !renderer.Ready() {
		return evidence.RenderResult{}, fault.New(fault.CodeInvalidInput, "evidenceapp.render", "renderer", "renderer is required")
	}
	if strings.TrimSpace(pdfPath) == "" || strings.TrimSpace(resultPath) == "" {
		return evidence.RenderResult{}, fault.New(fault.CodeInvalidInput, "evidenceapp.render", "output", "PDF and result paths are required")
	}
	if strings.TrimSpace(inputContract) == "" {
		return evidence.RenderResult{}, fault.New(fault.CodeInvalidInput, "evidenceapp.render", "input_contract", "input contract is required")
	}
	if strings.TrimSpace(view) == "" {
		return evidence.RenderResult{}, fault.New(fault.CodeInvalidInput, "evidenceapp.render", "view", "report view is required")
	}
	model := evidence.Canonicalize(admitted.Model)
	if err := evidence.Validate(ctx, model, limits); err != nil {
		return evidence.RenderResult{}, err
	}
	modelDigest, err := evidence.ModelDigest(model)
	if err != nil {
		return evidence.RenderResult{}, err
	}
	renderRequestDigest, err := evidence.RenderRequestDigestForView(model, view)
	if err != nil {
		return evidence.RenderResult{}, err
	}
	document, err := renderer.Render(ctx, model)
	if err != nil {
		return evidence.RenderResult{}, err
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return evidence.RenderResult{}, fault.Wrap(fault.CodeCanceled, "evidenceapp.render", ctxErr)
	}
	if len(document.Bytes) == 0 || len(document.Bytes) > maxPDFBytes || document.Pages < 1 || document.Pages > 500 || document.MediaType != "application/pdf" {
		return evidence.RenderResult{}, fault.New(fault.CodeRenderFailed, "evidenceapp.render", "document", "renderer returned an invalid or out-of-bounds PDF descriptor")
	}
	result := evidence.RenderResult{
		SchemaVersion: evidence.RenderResultVersion, ReportID: model.Identity.ReportID,
		ContractVersion: evidence.SchemaVersion, InputContract: inputContract,
		InputProfile: inputProfile, InputProfileVersion: inputProfileVersion,
		ReportProfile: reportProfile, ReportProfileVersion: reportProfileVersion, View: view,
		GeneratedAt: model.Identity.GeneratedAt.UTC(), AdmissionRequestSHA256: admitted.RequestBytesSHA256,
		ModelSHA256: modelDigest, RenderRequestSHA256: renderRequestDigest,
		PDF:            evidence.PDFDescriptor{Path: filepath.Base(pdfPath), MediaType: document.MediaType, SHA256: evidence.DigestBytes(document.Bytes), Bytes: len(document.Bytes), Pages: document.Pages},
		Generator:      evidence.BuildIdentity{Name: "breachsafe-pdf", Version: build.GeneratorVersion, Commit: build.GeneratorCommit},
		Renderer:       evidence.BuildIdentity{Name: document.RendererName, Version: document.RendererVersion},
		FontBundle:     evidence.BundleIdentity{Name: document.FontBundleName, SHA256: document.FontBundleSHA256},
		AssetBundle:    evidence.BundleIdentity{Name: "BreachSAFE approved report visuals/v1", SHA256: document.AssetBundleSHA256},
		InputArtifacts: model.Artifacts, CapabilityResults: document.Capabilities,
		Warnings: capabilityWarnings(document.Capabilities),
	}
	resultBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return evidence.RenderResult{}, fault.Wrap(fault.CodeRenderFailed, "evidenceapp.result", err)
	}
	resultBytes = append(resultBytes, '\n')
	if err := output.WritePair(ctx, output.Pair{PDFPath: pdfPath, PDFBytes: document.Bytes, ResultPath: resultPath, ResultBytes: resultBytes}); err != nil {
		return evidence.RenderResult{}, err
	}
	return result, nil
}

func readRegularFile(ctx context.Context, path string, maximum int, field string) (data []byte, retErr error) {
	if err := ctx.Err(); err != nil {
		return nil, fault.Wrap(fault.CodeCanceled, "evidenceapp.read", err)
	}
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return nil, fault.WrapField(fault.CodeInvalidInput, "evidenceapp.lstat", field, err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.Mode().IsRegular() {
		return nil, fault.New(fault.CodeInvalidInput, "evidenceapp.lstat", field, "input path must name a regular file directly")
	}
	// #nosec G304 -- the CLI explicitly accepts caller-selected local input paths;
	// Lstat rejects symlinks and non-regular files before this open.
	file, err := os.Open(path)
	if err != nil {
		return nil, fault.WrapField(fault.CodeInvalidInput, "evidenceapp.open", field, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && retErr == nil {
			data = nil
			retErr = fault.WrapField(fault.CodeInvalidInput, "evidenceapp.close", field, closeErr)
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, fault.WrapField(fault.CodeInvalidInput, "evidenceapp.stat", field, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fault.New(fault.CodeInvalidInput, "evidenceapp.stat", field, "input must resolve to a regular file")
	}
	if info.Size() < 1 || info.Size() > int64(maximum) {
		return nil, fault.Format(fault.CodeLimitExceeded, "evidenceapp.stat", field, "got %d bytes; maximum is %d", info.Size(), maximum)
	}
	data, err = io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, fault.WrapField(fault.CodeInvalidInput, "evidenceapp.read", field, err)
	}
	if len(data) > maximum {
		return nil, fault.Format(fault.CodeLimitExceeded, "evidenceapp.read", field, "input grew beyond maximum of %d bytes", maximum)
	}
	if err := ctx.Err(); err != nil {
		return nil, fault.Wrap(fault.CodeCanceled, "evidenceapp.read", err)
	}
	return data, nil
}

func capabilityWarnings(capabilities []evidence.CapabilityResult) []string {
	warnings := make([]string, 0)
	for _, capability := range capabilities {
		if capability.Status == "not_supported" {
			warnings = append(warnings, capability.Name+": "+capability.Detail)
		}
	}
	return warnings
}
