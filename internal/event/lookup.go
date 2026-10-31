package event

import (
	"errors"
	"strings"
)

var ErrInvalidIDs = errors.New("ids must contain 1 to 100 non-empty strings of at most 128 bytes")

type LookupResult struct {
	Events  []Event  `json:"events"`
	Missing []string `json:"missing"`
}

func (s *Service) Lookup(ids []string) (LookupResult, error) {
	result := LookupResult{Events: make([]Event, 0), Missing: make([]string, 0)}
	if len(ids) == 0 || len(ids) > 100 {
		return result, ErrInvalidIDs
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || len(id) > 128 {
			return result, ErrInvalidIDs
		}
	}
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		e, err := s.store.Get(id)
		if errors.Is(err, ErrNotFound) {
			result.Missing = append(result.Missing, id)
		} else if err != nil {
			return LookupResult{}, err
		} else {
			result.Events = append(result.Events, e)
		}
	}
	return result, nil
}
