// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package report

import (
	"context"
	"testing"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

func TestCommunityDocumentRoundTrip(t *testing.T) {
	model := evidence.CommunitySingleScan{SchemaVersion: evidence.SchemaVersion}
	document, err := NewCommunityDocument(model)
	if err != nil {
		t.Fatalf("NewCommunityDocument: %v", err)
	}
	if document.View != "community_single_scan" {
		t.Fatalf("view = %q", document.View)
	}
	decoded, err := document.CommunityModel()
	if err != nil {
		t.Fatalf("CommunityModel: %v", err)
	}
	if decoded.SchemaVersion != model.SchemaVersion {
		t.Fatalf("schema version = %q", decoded.SchemaVersion)
	}
}

var _ Builder = communityBuilder{}

type communityBuilder struct{}

func (communityBuilder) Build(context.Context, evidence.CommunitySingleScan, evidence.Limits) (Document, error) {
	return Document{}, nil
}
