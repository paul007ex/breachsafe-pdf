// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package manifest defines the versioned run-manifest input contract.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const SchemaVersion = "breachsafe.run.manifest/v1"

type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

type Run struct {
	Schema        string              `json:"schema"`
	InputProfile  string              `json:"input_profile"`
	ReportProfile string              `json:"report_profile,omitempty"`
	Request       Artifact            `json:"request"`
	Artifacts     map[string]Artifact `json:"artifacts"`
}

func Load(path string) (Run, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Run{}, fmt.Errorf("read manifest: %w", err)
	}
	var run Run
	if err := json.Unmarshal(data, &run); err != nil {
		return Run{}, fmt.Errorf("decode manifest: %w", err)
	}
	if run.Schema != SchemaVersion {
		return Run{}, fmt.Errorf("unsupported manifest schema %q", run.Schema)
	}
	if strings.TrimSpace(run.InputProfile) == "" {
		return Run{}, fmt.Errorf("manifest input_profile is required")
	}
	if strings.TrimSpace(run.Request.Path) == "" {
		return Run{}, fmt.Errorf("manifest request.path is required")
	}
	for _, name := range []string{"cbom", "scan-json"} {
		if strings.TrimSpace(run.Artifacts[name].Path) == "" {
			return Run{}, fmt.Errorf("manifest artifacts.%s.path is required", name)
		}
	}
	base := filepath.Dir(path)
	run.Request.Path = resolvePath(base, run.Request.Path)
	for name, artifact := range run.Artifacts {
		artifact.Path = resolvePath(base, artifact.Path)
		run.Artifacts[name] = artifact
	}
	return run, nil
}

func resolvePath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}
