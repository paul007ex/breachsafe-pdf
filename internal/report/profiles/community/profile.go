// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package community

import (
	"context"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/report"
)

const (
	ProfileID      = "breachsafe/community"
	ProfileVersion = "v1alpha1"
	View           = "community_single_scan"
)

type Profile struct{}

func (Profile) ID() string             { return ProfileID }
func (Profile) Version() string        { return ProfileVersion }
func (Profile) InputAdapterID() string { return "qureddy-single-scan" }
func (Profile) View() string           { return View }

func (Profile) Validate(ctx context.Context, model evidence.CommunitySingleScan, limits evidence.Limits) error {
	return evidence.Validate(ctx, model, limits)
}

func (Profile) Build(ctx context.Context, model evidence.CommunitySingleScan, limits evidence.Limits) (report.Document, error) {
	if err := evidence.Validate(ctx, model, limits); err != nil {
		return report.Document{}, err
	}
	return report.NewCommunityDocument(model)
}
