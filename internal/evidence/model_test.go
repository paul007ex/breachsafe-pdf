// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package evidence_test

import (
	"context"
	"os"
	"testing"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

func TestAdmittedModelPassesClosedContract(t *testing.T) {
	model := validModel(t)
	if err := evidence.Validate(context.Background(), model, evidence.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestClosedEnumsReferencesAndTextFailInvalidModels(t *testing.T) {
	tests := map[string]func(*evidence.CommunitySingleScan){
		"finding severity":      func(model *evidence.CommunitySingleScan) { model.Findings.Items[0].Severity = "favorable" },
		"finding readiness":     func(model *evidence.CommunitySingleScan) { model.Findings.Items[0].Readiness = "secure" },
		"inventory observation": func(model *evidence.CommunitySingleScan) { model.Inventory.Items[0].Observation = "guessed" },
		"tool state":            func(model *evidence.CommunitySingleScan) { model.Tools[0].State = "green" },
		"dangling source":       func(model *evidence.CommunitySingleScan) { model.PostureAxes[0].SourceRefs = []string{"absent"} },
		"collection accounting": func(model *evidence.CommunitySingleScan) { model.Findings.TotalCount++ },
		"bidi override":         func(model *evidence.CommunitySingleScan) { model.Inventory.Items[0].Name += "\u202eexe" },
		"duplicate limitation": func(model *evidence.CommunitySingleScan) {
			model.Limitations = append(model.Limitations, model.Limitations[0])
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			model := validModel(t)
			mutate(&model)
			if err := evidence.Validate(context.Background(), model, evidence.DefaultLimits()); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func validModel(t testing.TB) evidence.CommunitySingleScan {
	t.Helper()
	request := mustRead(t, "../../examples/community-single-scan.request.json")
	cbom := mustRead(t, "../admission/testdata/success.cbom.json")
	scan := mustRead(t, "../admission/testdata/success.scan.json")
	result, err := admission.Admit(context.Background(), request, cbom, scan, admission.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return result.Model
}

func mustRead(t testing.TB, path string) []byte {
	t.Helper()
	// #nosec G304 -- test fixtures are fixed repository paths, not user input.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
