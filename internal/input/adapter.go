// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package input contains producer-facing admission boundaries. Adapters turn
// exact producer bytes into the source-neutral evidence model; renderers never
// depend on producer wire formats.
package input

import (
	"context"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
)

// Input contains the exact bytes supplied to an adapter. The map is keyed by
// stable artifact role (for example, "cbom" and "scan-json").
type Input struct {
	Request   []byte
	Artifacts map[string][]byte
}

// Admission identifies the adapter that admitted the exact source bytes.
type Admission struct {
	Result   admission.Result
	ID       string
	Version  string
	Contract string
}

// Adapter validates producer-native inputs and projects them into the
// normalized report model.
type Adapter interface {
	ID() string
	Version() string
	Admit(context.Context, Input, admission.Limits) (Admission, error)
}
