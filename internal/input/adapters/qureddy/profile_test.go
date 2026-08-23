// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package qureddy

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
	"github.com/paul007ex/breachsafe-pdf/internal/input"
)

func TestAdapterPreservesStrictAdmission(t *testing.T) {
	request, err := os.ReadFile("../../../../examples/community-single-scan.request.json")
	if err != nil {
		t.Fatal(err)
	}
	cbom, err := os.ReadFile("../../../admission/testdata/success.cbom.json")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := os.ReadFile("../../../admission/testdata/success.scan.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := (Adapter{}).Admit(context.Background(), input.Input{Request: request, Artifacts: map[string][]byte{"cbom": cbom, "scan-json": scan}}, admission.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != AdapterID || result.Version != AdapterVersion || result.Result.Model.SchemaVersion == "" {
		t.Fatalf("adapter result = %#v", result)
	}
}

func TestAdapterRejectsMissingArtifact(t *testing.T) {
	_, err := (Adapter{}).Admit(context.Background(), input.Input{}, admission.DefaultLimits())
	var typed *fault.Error
	if !errors.As(err, &typed) || typed.Code != fault.CodeInvalidInput {
		t.Fatalf("error = %v", err)
	}
}
