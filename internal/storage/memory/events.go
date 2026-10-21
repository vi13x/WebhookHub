package memory

import (
	"bytes"
	"sync"

	"webhookhub/internal/event"
)

type EventStore struct {
	mu     sync.RWMutex
	events map[string]event.Event
	ids    []string
	keys   map[string]string
}

func NewEventStore() *EventStore {
	return &EventStore{events: make(map[string]event.Event), keys: make(map[string]string)}
}

func (s *EventStore) Save(e event.Event, key string) (event.Event, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key != "" {
		if id, exists := s.keys[key]; exists {
			existing := s.events[id]
			if existing.Type != e.Type || !bytes.Equal(existing.Data, e.Data) {
				return event.Event{}, false, event.ErrConflict
			}
			return clone(existing), false, nil
		}
		s.keys[key] = e.ID
	}
	if _, exists := s.events[e.ID]; !exists {
		s.ids = append(s.ids, e.ID)
	}
	s.events[e.ID] = clone(e)
	return clone(e), true, nil
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
