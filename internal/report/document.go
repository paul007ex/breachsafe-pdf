// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package report

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
)

// Document is the neutral hand-off between a report profile and a document
// writer. Payload is intentionally opaque to the writer; each profile owns its
// schema and view contract.
type Document struct {
	View    string          `json:"view"`
	Version string          `json:"version"`
	Payload json.RawMessage `json:"payload"`
}

func NewCommunityDocument(model evidence.CommunitySingleScan) (Document, error) {
	payload, err := json.Marshal(model)
	if err != nil {
		return Document{}, fmt.Errorf("encode community document: %w", err)
	}
	return Document{View: "community_single_scan", Version: "v1alpha1", Payload: payload}, nil
}

func (document Document) CommunityModel() (evidence.CommunitySingleScan, error) {
	if document.View != "community_single_scan" {
		return evidence.CommunitySingleScan{}, fmt.Errorf("unsupported document view %q", document.View)
	}
	var model evidence.CommunitySingleScan
	if err := json.Unmarshal(document.Payload, &model); err != nil {
		return evidence.CommunitySingleScan{}, fmt.Errorf("decode community document: %w", err)
	}
	return model, nil
}

// Builder projects normalized evidence into a report document.
type Builder interface {
	Build(context.Context, evidence.CommunitySingleScan, evidence.Limits) (Document, error)
}
