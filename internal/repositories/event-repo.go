package repositories

import (
	"errors"
	"fmt"
	"sync"
	. "api-aprop/internal/models"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrInvalidEvent  = errors.New("invalid event")
)

type EventStore interface {
	Create(title string, description string) (Event, error)
	Get(id int64) (Event, error)
	List() []Event
}

type memoryStore struct {
	mu     sync.RWMutex
	events map[int64]Event
	nextID int64
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{events: make(map[int64]Event), nextID: 1}
}

func (s *memoryStore) Create(title string, description string) (Event, error) {
	if title == "" {
		return Event{}, fmt.Errorf("create event: %w", ErrInvalidEvent)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t := Event{ID: s.nextID, Title: title, Description: description}
	s.events[t.ID] = t
	s.nextID++
	return t, nil
}

func (s *memoryStore) Get(id int64) (Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.events[id]
	if !ok {
		return Event{}, fmt.Errorf("get event %d: %w", id, ErrEventNotFound)
	}
	return t, nil
}

func (s *memoryStore) List() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, 0, len(s.events))
	for _, t := range s.events {
		out = append(out, t)
	}
	return out
}