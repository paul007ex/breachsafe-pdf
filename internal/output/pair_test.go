// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package output

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/paul007ex/breachsafe-pdf/internal/fault"
)

func TestWritePairCreatesExactPrivateOutputs(t *testing.T) {
	directory := t.TempDir()
	pair := Pair{
		PDFPath: filepath.Join(directory, "report.pdf"), PDFBytes: []byte("pdf-bytes"),
		ResultPath: filepath.Join(directory, "report.json"), ResultBytes: []byte("json-bytes"),
	}
	if err := WritePair(context.Background(), pair); err != nil {
		t.Fatal(err)
	}
	for path, expected := range map[string]string{pair.PDFPath: "pdf-bytes", pair.ResultPath: "json-bytes"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != expected {
			t.Fatalf("%s = %q, want %q", path, data, expected)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o, want 600", path, info.Mode().Perm())
		}
	}
}

func TestWritePairRefusesClobberAndRollsBackCreatedPDF(t *testing.T) {
	directory := t.TempDir()
	pdfPath := filepath.Join(directory, "report.pdf")
	resultPath := filepath.Join(directory, "report.json")
	if err := os.WriteFile(resultPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := WritePair(context.Background(), Pair{
		PDFPath: pdfPath, PDFBytes: []byte("new-pdf"),
		ResultPath: resultPath, ResultBytes: []byte("new-json"),
	})
	if err == nil || fault.CodeOf(err) != fault.CodeOutputExists {
		t.Fatalf("expected OUTPUT_EXISTS, got %v", err)
	}
	if _, err := os.Stat(pdfPath); !os.IsNotExist(err) {
		t.Fatalf("new PDF was not rolled back: %v", err)
	}
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("existing result changed: %q", data)
	}
}

func TestWritePairHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := WritePair(ctx, Pair{PDFPath: "x", PDFBytes: []byte("x"), ResultPath: "y", ResultBytes: []byte("y")})
	if err == nil || fault.CodeOf(err) != fault.CodeCanceled {
		t.Fatalf("expected CANCELED, got %v", err)
	}
}
