package memory

import (
	"sync"

	"webhookhub/internal/event"
)

type EventStore struct {
	mu     sync.RWMutex
	events map[string]event.Event
	ids    []string
}

func NewEventStore() *EventStore {
	return &EventStore{events: make(map[string]event.Event)}
}

func (s *EventStore) Save(e event.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.events[e.ID]; !exists {
		s.ids = append(s.ids, e.ID)
	}
	s.events[e.ID] = clone(e)
}

func (s *EventStore) List() []event.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]event.Event, 0, len(s.ids))
	for _, id := range s.ids {
		result = append(result, clone(s.events[id]))
	}
	return result
}

func (s *EventStore) Get(id string) (event.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.events[id]
	if !ok {
		return event.Event{}, event.ErrNotFound
	}
	return clone(e), nil
}

func clone(e event.Event) event.Event {
	e.Data = append([]byte(nil), e.Data...)
	return e
}
