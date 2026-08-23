// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package report contains audience-facing report profile contracts. A report
// profile consumes normalized evidence and never parses producer-native bytes.
package report

import (
	"context"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

// Profile defines one stable report projection.
type Profile interface {
	ID() string
	Version() string
	InputAdapterID() string
	View() string
	Validate(context.Context, evidence.CommunitySingleScan, evidence.Limits) error
	Build(context.Context, evidence.CommunitySingleScan, evidence.Limits) (Document, error)
}
