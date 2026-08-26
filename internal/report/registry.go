// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package report

import (
	"fmt"
	"sort"
)

// Registry resolves a report profile by its identifier.
type Registry struct {
	profiles map[string]Profile
}

// NewRegistry builds a Registry from the given profiles. It returns an error on a
// duplicate identifier rather than silently keeping the last one.
func NewRegistry(profiles ...Profile) (Registry, error) {
	result := Registry{profiles: make(map[string]Profile, len(profiles))}
	for _, profile := range profiles {
		if profile == nil {
			return Registry{}, fmt.Errorf("report profile is nil")
		}
		if _, exists := result.profiles[profile.ID()]; exists {
			return Registry{}, fmt.Errorf("duplicate report profile %q", profile.ID())
		}
		result.profiles[profile.ID()] = profile
	}
	return result, nil
}

// Resolve returns the profile registered under id, or an error naming what was asked for.
func (r Registry) Resolve(id string) (Profile, error) {
	profile, ok := r.profiles[id]
	if !ok {
		return nil, fmt.Errorf("unsupported report profile %q", id)
	}
	return profile, nil
}

// IDs returns every registered profile identifier.
func (r Registry) IDs() []string {
	ids := make([]string, 0, len(r.profiles))
	for id := range r.profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
