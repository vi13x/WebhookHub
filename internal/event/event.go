package event

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound   = errors.New("event not found")
	ErrInvalid    = errors.New("type and non-null JSON data are required")
	ErrConflict   = errors.New("idempotency key already used with different event data")
	ErrInvalidKey = errors.New("idempotency key must contain 1 to 200 visible ASCII characters")
)

type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

type Store interface {
	Save(Event, string) (Event, bool, error)
	List() []Event
	Get(string) (Event, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(kind string, data json.RawMessage, key string) (Event, bool, error) {
	if len(key) > 200 {
		return Event{}, false, ErrInvalidKey
	}
	for _, ch := range key {
		if ch < 33 || ch > 126 {
			return Event{}, false, ErrInvalidKey
		}
	}
	kind = strings.TrimSpace(kind)
	if kind == "" || !json.Valid(data) || strings.TrimSpace(string(data)) == "null" {
		return Event{}, false, ErrInvalid
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return Event{}, false, ErrInvalid
	}
	data, err := json.Marshal(value)
	if err != nil {
		return Event{}, false, err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Event{}, false, err
	}
	e := Event{
		ID: hex.EncodeToString(id[:]), Type: kind,
		Data:   append(json.RawMessage(nil), data...),
		Status: "pending", CreatedAt: time.Now().UTC(),
	}
	return s.store.Save(e, key)
}

func (s *Service) List() []Event { return s.store.List() }

func (s *Service) Get(id string) (Event, error) { return s.store.Get(id) }
