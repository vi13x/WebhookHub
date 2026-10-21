package event

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("event not found")
	ErrInvalid  = errors.New("type and non-null JSON data are required")
)

type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

type Store interface {
	Save(Event)
	List() []Event
	Get(string) (Event, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(kind string, data json.RawMessage) (Event, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" || !json.Valid(data) || strings.TrimSpace(string(data)) == "null" {
		return Event{}, ErrInvalid
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Event{}, err
	}
	e := Event{
		ID: hex.EncodeToString(id[:]), Type: kind,
		Data:   append(json.RawMessage(nil), data...),
		Status: "pending", CreatedAt: time.Now().UTC(),
	}
	s.store.Save(e)
	return e, nil
}

func (s *Service) List() []Event { return s.store.List() }

func (s *Service) Get(id string) (Event, error) { return s.store.Get(id) }
