// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	data := []byte(`{"schema":"breachsafe.run.manifest/v1","input_profile":"qureddy-single-scan","report_profile":"breachsafe/community","request":{"path":"request.json"},"artifacts":{"cbom":{"path":"cbom.json"},"scan-json":{"path":"scan.json"}}}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	run, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if run.InputProfile != "qureddy-single-scan" || run.Artifacts["cbom"].Path != filepath.Join(filepath.Dir(path), "cbom.json") {
		t.Fatalf("unexpected manifest: %+v", run)
	}
}

func TestLoadRejectsUnsupportedSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	if err := os.WriteFile(path, []byte(`{"schema":"future/v2"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("unsupported schema accepted")
	}
}
