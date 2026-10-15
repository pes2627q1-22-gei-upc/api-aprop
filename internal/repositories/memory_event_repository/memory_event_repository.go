package memory_event_repository

import (
	"fmt"
	"sync"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/event_repository"
)

// MemoryEventRepository keeps events in memory. Data is lost when the server stops.
type MemoryEventRepository struct {
	mu     sync.RWMutex
	events map[int]event_model.Event
	nextID int
}

func NewMemoryEventRepository() *MemoryEventRepository {
	return &MemoryEventRepository{
		events: make(map[int]event_model.Event),
		nextID: 1,
	}
}

// Create stores the event. If its ID does not exist yet, a new one is assigned.
func (r *MemoryEventRepository) Create(e event_model.Event) (event_model.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.events[e.ID]; !exists {
		e.ID = r.nextID
		r.nextID++
	}
	r.events[e.ID] = e
	return e, nil
}

func (r *MemoryEventRepository) Get(id int) (event_model.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.events[id]
	if !ok {
		return event_model.Event{}, fmt.Errorf("get event %d: %w", id, event_repository.ErrEventNotFound)
	}
	return e, nil
}

func (r *MemoryEventRepository) List() []event_model.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]event_model.Event, 0, len(r.events))
	for _, e := range r.events {
		out = append(out, e)
	}
	return out
}
