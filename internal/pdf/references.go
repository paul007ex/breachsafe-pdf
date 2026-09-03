// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package pdf

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed references.json
var referencesJSON []byte

// referenceRule maps a finding rule-id substring to its standards references and CWE.
type referenceRule struct {
	Match []string `json:"match"`
	CWE   string   `json:"cwe"`
	Refs  []string `json:"refs"`
	Fix   string   `json:"fix"`
}

// referenceResource is one external source listed on the references page.
type referenceResource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type referenceCatalog struct {
	Rules     []referenceRule     `json:"rules"`
	Resources []referenceResource `json:"resources"`
}

var references = loadReferences()

func loadReferences() referenceCatalog {
	var catalog referenceCatalog
	// The catalog is a build-time embedded asset; a parse failure is a programming
	// error, so fall back to an empty catalog rather than panicking a render.
	_ = json.Unmarshal(referencesJSON, &catalog)
	return catalog
}

// lookup returns the references, CWE, and fix for a finding rule id. Rules are
// evaluated in order, so specific matches must precede general ones in the JSON.
func (c referenceCatalog) lookup(ruleRef string) (refs []string, cwe, fix string) {
	lower := strings.ToLower(ruleRef)
	for _, rule := range c.Rules {
		for _, token := range rule.Match {
			if strings.Contains(lower, strings.ToLower(token)) {
				return rule.Refs, rule.CWE, rule.Fix
			}
		}
	}
	return nil, "", ""
}
