// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package input

import (
	"fmt"
	"sort"
)

// Registry resolves explicit, versioned input adapter IDs.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry constructs a registry and rejects duplicate IDs.
func NewRegistry(adapters ...Adapter) (Registry, error) {
	result := Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		if adapter == nil {
			return Registry{}, fmt.Errorf("input adapter is nil")
		}
		if _, exists := result.adapters[adapter.ID()]; exists {
			return Registry{}, fmt.Errorf("duplicate input adapter %q", adapter.ID())
		}
		result.adapters[adapter.ID()] = adapter
	}
	return result, nil
}

// Resolve returns an explicitly named adapter.
func (r Registry) Resolve(id string) (Adapter, error) {
	adapter, ok := r.adapters[id]
	if !ok {
		return nil, fmt.Errorf("unsupported input adapter %q", id)
	}
	return adapter, nil
}

// IDs returns deterministic registry contents for CLI discovery.
func (r Registry) IDs() []string {
	ids := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
