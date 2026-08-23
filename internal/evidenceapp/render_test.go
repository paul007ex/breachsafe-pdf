// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package evidenceapp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
	"github.com/paul007ex/breachsafe-pdf/internal/pdf"
)

func TestRenderFilesWritesExactNoClobberPair(t *testing.T) {
	directory := t.TempDir()
	pdfPath := filepath.Join(directory, "report.pdf")
	resultPath := filepath.Join(directory, "report.result.json")
	request := FileRequest{
		RequestPath:  "../../examples/community-single-scan.request.json",
		CBOMPath:     "../admission/testdata/success.cbom.json",
		ScanJSONPath: "../admission/testdata/success.scan.json",
		PDFPath:      pdfPath, ResultPath: resultPath,
	}
	build := Build{GeneratorVersion: "test", GeneratorCommit: "deadbeef"}
	result, err := RenderFiles(context.Background(), request, pdf.New("test", "deadbeef"), admission.DefaultLimits(), build)
	if err != nil {
		t.Fatal(err)
	}
	pdfBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.PDF.SHA256 != evidence.DigestBytes(pdfBytes) || result.PDF.Bytes != len(pdfBytes) || result.PDF.Pages != 9 {
		t.Fatalf("result PDF descriptor = %+v", result.PDF)
	}
	resultBytes, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var persisted evidence.RenderResult
	if err := json.Unmarshal(resultBytes, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.PDF.SHA256 != result.PDF.SHA256 || len(persisted.Warnings) != 4 || len(persisted.AdmissionRequestSHA256) != 64 {
		t.Fatalf("persisted result = %+v", persisted)
	}
	for _, path := range []string{pdfPath, resultPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o", path, info.Mode().Perm())
		}
	}
	_, err = RenderFiles(context.Background(), request, pdf.New("test", "deadbeef"), admission.DefaultLimits(), build)
	assertFaultCode(t, err, fault.CodeOutputExists)
}

func TestReadRegularFileRejectsSymlinkAndDirectory(t *testing.T) {
	directory := t.TempDir()
	regular := filepath.Join(directory, "input.json")
	if err := os.WriteFile(regular, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(directory, "input-link.json")
	if err := os.Symlink(regular, symlink); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{symlink, directory} {
		_, err := readRegularFile(context.Background(), path, 1024, "test")
		assertFaultCode(t, err, fault.CodeInvalidInput)
	}
}

func TestRenderRejectsTypedNilRenderer(t *testing.T) {
	admitted := admittedResult(t)
	var renderer *stubRenderer
	_, err := Render(context.Background(), admitted, filepath.Join(t.TempDir(), "report.pdf"), filepath.Join(t.TempDir(), "result.json"), renderer, evidence.DefaultLimits(), Build{})
	assertFaultCode(t, err, fault.CodeInvalidInput)
}

func TestRenderRejectsIncompleteRendererDescriptor(t *testing.T) {
	admitted := admittedResult(t)
	_, err := Render(context.Background(), admitted, filepath.Join(t.TempDir(), "report.pdf"), filepath.Join(t.TempDir(), "result.json"), &stubRenderer{}, evidence.DefaultLimits(), Build{})
	assertFaultCode(t, err, fault.CodeRenderFailed)
}

func TestRenderFilesHonorsCanceledContextBeforeRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RenderFiles(ctx, FileRequest{
		RequestPath:  "../../examples/community-single-scan.request.json",
		CBOMPath:     "../admission/testdata/success.cbom.json",
		ScanJSONPath: "../admission/testdata/success.scan.json",
		PDFPath:      filepath.Join(t.TempDir(), "report.pdf"), ResultPath: filepath.Join(t.TempDir(), "result.json"),
	}, pdf.New("test", ""), admission.DefaultLimits(), Build{})
	assertFaultCode(t, err, fault.CodeCanceled)
}

type stubRenderer struct{}

func (renderer *stubRenderer) Ready() bool { return renderer != nil }

func (*stubRenderer) Render(context.Context, evidence.CommunitySingleScan) (evidence.Document, error) {
	return evidence.Document{}, nil
}

func admittedResult(t *testing.T) admission.Result {
	t.Helper()
	request, err := os.ReadFile("../../examples/community-single-scan.request.json")
	if err != nil {
		t.Fatal(err)
	}
	cbom, err := os.ReadFile("../admission/testdata/success.cbom.json")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := os.ReadFile("../admission/testdata/success.scan.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := admission.Admit(context.Background(), request, cbom, scan, admission.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertFaultCode(t *testing.T, err error, want fault.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %s", want)
	}
	var typed *fault.Error
	if !errors.As(err, &typed) || typed.Code != want {
		t.Fatalf("error = %v, want code %s", err, want)
	}
}
