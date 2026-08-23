// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestHelpCommandsAreSuccessful(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"profile", "--help"},
		{"profile", "render", "--help"},
		{"render", "--help"},
		{"help", "profile", "render"},
	} {
		if code := run(args); code != 0 {
			t.Errorf("run(%q) = %d, want 0", args, code)
		}
	}
}

func TestHelpSubcommandsRenderTheirOwnUsage(t *testing.T) {
	for _, test := range []struct {
		args   []string
		needle string
	}{
		{args: []string{"help", "profile"}, needle: "usage: breachsafe-pdf profile"},
		{args: []string{"help", "render"}, needle: "usage: breachsafe-pdf render"},
	} {
		output, code := captureStdout(func() int { return run(test.args) })
		if code != 0 {
			t.Errorf("run(%q) = %d, want 0", test.args, code)
		}
		if !strings.Contains(output, test.needle) {
			t.Errorf("run(%q) output %q does not contain %q", test.args, output, test.needle)
		}
	}
}

func captureStdout(fn func() int) (string, int) {
	original := os.Stdout
	reader, writer, _ := os.Pipe()
	os.Stdout = writer
	code := fn()
	_ = writer.Close()
	os.Stdout = original
	output, _ := io.ReadAll(reader)
	_ = reader.Close()
	return string(output), code
}

func TestLoggerFormats(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		if _, err := newLogger(format, false); err != nil {
			t.Fatalf("newLogger(%q): %v", format, err)
		}
	}
	if _, err := newLogger("xml", false); err == nil {
		t.Fatal("unsupported logger format accepted")
	}
}

func TestUnknownProfileFailsClosed(t *testing.T) {
	if code := run([]string{"profile", "inspect", "breachsafe/unknown"}); code == 0 {
		t.Fatal("unknown profile accepted")
	}
}

func TestRenderPreservesTypedInputExitCode(t *testing.T) {
	code := run([]string{
		"render", "--profile", "breachsafe/community",
		"--request", "/tmp/breachsafe-pdf-test-missing-request.json",
		"--cbom", "/tmp/breachsafe-pdf-test-missing-cbom.json",
		"--scan-json", "/tmp/breachsafe-pdf-test-missing-scan.json",
		"--pdf", "/tmp/breachsafe-pdf-test-output.pdf",
		"--result", "/tmp/breachsafe-pdf-test-result.json",
	})
	if code != 2 {
		t.Fatalf("render invalid input exit code = %d, want 2", code)
	}
}

func TestParseRenderOptionsSupportsExplicitProfiles(t *testing.T) {
	options, err := parseRenderOptions([]string{
		"--input-profile", "qureddy-single-scan",
		"--report-profile", "breachsafe/community",
		"--request", "request.json",
		"--cbom", "scan.cdx.json",
		"--scan-json", "scan.json",
	})
	if err != nil {
		t.Fatalf("parseRenderOptions: %v", err)
	}
	if options.inputProfileID != "qureddy-single-scan" {
		t.Fatalf("input profile = %q", options.inputProfileID)
	}
	if options.reportProfileID != "breachsafe/community" {
		t.Fatalf("report profile = %q", options.reportProfileID)
	}
}

func TestParseRenderOptionsPreservesLegacyProfileAlias(t *testing.T) {
	options, err := parseRenderOptions([]string{
		"--profile", "breachsafe/community",
		"--request", "request.json",
		"--cbom", "scan.cdx.json",
		"--scan-json", "scan.json",
	})
	if err != nil {
		t.Fatalf("parseRenderOptions: %v", err)
	}
	if options.reportProfileID != "breachsafe/community" {
		t.Fatalf("report profile = %q", options.reportProfileID)
	}
}

func TestParseRenderOptionsRejectsConflictingReportProfiles(t *testing.T) {
	if _, err := parseRenderOptions([]string{
		"--profile", "breachsafe/community",
		"--report-profile", "breachsafe/community",
		"--request", "request.json",
		"--cbom", "scan.cdx.json",
		"--scan-json", "scan.json",
	}); err == nil {
		t.Fatal("legacy and explicit report profiles accepted together")
	}
}
