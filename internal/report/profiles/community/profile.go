// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package community implements the community single-scan report profile.
package community

import (
	"context"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

// Identity of the community report profile, and the model view it renders.
const (
	ProfileID      = "breachsafe/community"
	ProfileVersion = "v1alpha1"
	View           = "community_single_scan"
)

// Profile is the community single-scan report profile.
type Profile struct{}

// ID returns the profile identifier.
func (Profile) ID() string { return ProfileID }

// Version returns the profile contract version.
func (Profile) Version() string { return ProfileVersion }

// InputAdapterID returns the input adapter this profile expects.
func (Profile) InputAdapterID() string { return "qureddy-single-scan" }

// View returns the model view this profile renders.
func (Profile) View() string { return View }

// Validate checks the model against the profile's limits before it is rendered.
func (Profile) Validate(ctx context.Context, model evidence.CommunitySingleScan, limits evidence.Limits) error {
	return evidence.Validate(ctx, model, limits)
}
