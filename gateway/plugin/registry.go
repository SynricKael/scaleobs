package plugin

import (
	"sort"
	"sync"

	"github.com/glrs/observer/gateway/model"
)

// Registry holds all registered services and provides lookup methods.
type Registry struct {
	mu       sync.RWMutex
	services map[string]*model.Service
}

// NewRegistry creates a new Registry from a service slice.
func NewRegistry(services []model.Service) *Registry {
	r := &Registry{
		services: make(map[string]*model.Service, len(services)),
	}
	for i := range services {
		s := services[i]
		r.services[s.ID] = &s
	}
	return r
}

// Get returns a service by ID. Returns nil if not found.
func (r *Registry) Get(id string) *model.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[id]
}

// List returns all services sorted by category then name.
func (r *Registry) List() []*model.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*model.Service, 0, len(r.services))
	for _, s := range r.services {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].Name < result[j].Name
	})
	return result
}

// ListByCategory returns services grouped by category.
func (r *Registry) ListByCategory() map[string][]*model.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groups := make(map[string][]*model.Service)
	for _, s := range r.services {
		groups[s.Category] = append(groups[s.Category], s)
	}
	// Sort each group
	for _, list := range groups {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Name < list[j].Name
		})
	}
	return groups
}

// Categories returns the unique categories in sorted order.
func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	for _, s := range r.services {
		seen[s.Category] = true
	}
	result := make([]string, 0, len(seen))
	for c := range seen {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

// Reload replaces the registry contents (for hot-reload).
func (r *Registry) Reload(services []model.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newMap := make(map[string]*model.Service, len(services))
	for i := range services {
		s := services[i]
		newMap[s.ID] = &s
	}
	r.services = newMap
}
