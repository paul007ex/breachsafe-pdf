// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package main

import "testing"

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
