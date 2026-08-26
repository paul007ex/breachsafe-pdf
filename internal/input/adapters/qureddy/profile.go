// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package qureddy is the first producer adapter. Its implementation delegates
// to the existing strict admission path so the profile migration cannot relax
// validation or correlation behavior.
package qureddy

import (
	"context"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
	"github.com/paul007ex/breachsafe-pdf/internal/input"
)

// Identity of this input adapter, carried in the emitted document.
const (
	AdapterID      = "qureddy-single-scan"
	AdapterVersion = "v1alpha1"
)

// Adapter admits a QuReddy single-scan artifact set as report input.
type Adapter struct{}

// ID returns the adapter identifier.
func (Adapter) ID() string { return AdapterID }

// Version returns the adapter contract version.
func (Adapter) Version() string { return AdapterVersion }

// Admit validates the supplied artifacts against limits and decodes them into an
// input.Admission. It returns a fault rather than a partial result on any failure.
func (Adapter) Admit(ctx context.Context, in input.Input, limits admission.Limits) (input.Admission, error) {
	cbom, ok := in.Artifacts["cbom"]
	if !ok {
		return input.Admission{}, admissionInputError("cbom")
	}
	scan, ok := in.Artifacts["scan-json"]
	if !ok {
		return input.Admission{}, admissionInputError("scan-json")
	}
	result, err := admission.Admit(ctx, in.Request, cbom, scan, limits)
	if err != nil {
		return input.Admission{}, err
	}
	return input.Admission{Result: result, ID: AdapterID, Version: AdapterVersion, Contract: evidence.InputContract}, nil
}

func admissionInputError(name string) error {
	return fault.New(fault.CodeInvalidInput, "input.qureddy", name, "required artifact is missing")
}
