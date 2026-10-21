package memory_test

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"webhookhub/internal/event"
	"webhookhub/internal/storage/memory"
)

func TestStoreOwnsData(t *testing.T) {
	s := memory.NewEventStore()
	e := event.Event{ID: "one", Data: json.RawMessage(`{"a":1}`)}
	s.Save(e)
	e.Data[0] = '!'
	got, _ := s.Get("one")
	got.Data[0] = '!'
	list := s.List()
	list[0].Data[0] = '!'
	got, _ = s.Get("one")
	if string(got.Data) != `{"a":1}` {
		t.Fatal("caller changed stored data")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := memory.NewEventStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			s.Save(event.Event{ID: id})
			if _, err := s.Get(id); err != nil {
				t.Error(err)
			}
			_ = s.List()
		}(fmt.Sprint(i))
	}
	wg.Wait()
	if len(s.List()) != 50 {
		t.Fatal("events were lost")
	}
}
