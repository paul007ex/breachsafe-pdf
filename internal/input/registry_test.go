// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package input

import (
	"context"
	"testing"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
)

type testAdapter string

func (a testAdapter) ID() string    { return string(a) }
func (testAdapter) Version() string { return "v1" }
func (testAdapter) Admit(context.Context, Input, admission.Limits) (Admission, error) {
	return Admission{}, nil
}

func TestRegistryRejectsDuplicateAndSortsIDs(t *testing.T) {
	registry, err := NewRegistry(testAdapter("z"), testAdapter("a"))
	if err != nil {
		t.Fatal(err)
	}
	ids := registry.IDs()
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "z" {
		t.Fatalf("ids = %#v", ids)
	}
	if _, err := NewRegistry(testAdapter("a"), testAdapter("a")); err == nil {
		t.Fatal("duplicate adapter accepted")
	}
}
