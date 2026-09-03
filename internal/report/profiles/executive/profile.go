// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package executive implements the one-page Harvest Now, Decrypt Later risk exposure
// summary report profile. It consumes the same admitted model as the community profile.
package executive

import (
	"context"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

// Identity of the executive report profile, and the model view it renders.
const (
	ProfileID      = "breachsafe/executive"
	ProfileVersion = "v1alpha1"
	View           = "executive_hndl_summary"
)

// Profile is the executive HNDL risk exposure summary report profile.
type Profile struct{}

// ID returns the profile identifier.
func (Profile) ID() string { return ProfileID }

// Version returns the profile contract version.
func (Profile) Version() string { return ProfileVersion }

// InputAdapterID returns the input adapter this profile expects. It reuses the same
// admitted model as the community profile.
func (Profile) InputAdapterID() string { return "qureddy-single-scan" }

// View returns the model view this profile renders.
func (Profile) View() string { return View }

// Validate checks the model against the profile's limits before it is rendered.
func (Profile) Validate(ctx context.Context, model evidence.CommunitySingleScan, limits evidence.Limits) error {
	return evidence.Validate(ctx, model, limits)
}
