package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"webhookhub/internal/event"
	"webhookhub/internal/httpapi"
	"webhookhub/internal/storage/memory"
)

func TestEventLifecycle(t *testing.T) {
	h := httpapi.NewHandler(event.NewService(memory.NewEventStore()))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/events", ""); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", w.Code, w.Body.String())
	}
	w := request("POST", "/events", `{"type":"order.created","data":{"order_id":42}}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created event.Event
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Status != "pending" || created.CreatedAt.IsZero() {
		t.Fatalf("invalid event: %+v", created)
	}
	if w.Header().Get("Location") != "/events/"+created.ID {
		t.Fatal("missing event location")
	}
	w = request("GET", "/events/"+created.ID, "")
	var fetched event.Event
	if err := json.Unmarshal(w.Body.Bytes(), &fetched); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || fetched.ID != created.ID || string(fetched.Data) != `{"order_id":42}` {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
	w = request("GET", "/events", "")
	var list []event.Event
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list: %s", w.Body.String())
	}
	if w := request("GET", "/events/missing", ""); w.Code != 404 {
		t.Fatalf("missing event: %d", w.Code)
	}
}

func TestInvalidEventsAreNotStored(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType string
		status                  int
	}{
		{"malformed", `{`, "application/json", 400},
		{"missing type", `{"data":{}}`, "application/json", 400},
		{"blank type", `{"type":"  ","data":{}}`, "application/json", 400},
		{"missing data", `{"type":"test"}`, "application/json", 400},
		{"null data", `{"type":"test","data":null}`, "application/json", 400},
		{"unknown field", `{"type":"test","data":{},"extra":1}`, "application/json", 400},
		{"multiple values", `{"type":"test","data":{}} {}`, "application/json", 400},
		{"null request", `null`, "application/json", 400},
		{"wrong media type", `{}`, "text/plain", 415},
		{"too large", `{"type":"test","data":"` + strings.Repeat("a", 1<<20) + `"}`, "application/json", 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := memory.NewEventStore()
			h := httpapi.NewHandler(event.NewService(store))
			r := httptest.NewRequest("POST", "/events", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || len(store.List()) != 0 {
				t.Fatalf("status=%d, body=%s, stored=%d", w.Code, w.Body.String(), len(store.List()))
			}
		})
	}
}

func TestIdempotency(t *testing.T) {
	store := memory.NewEventStore()
	h := httpapi.NewHandler(event.NewService(store))
	send := func(key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/events", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	body := `{"type":"order.created","data":{"order_id":42,"amount":1500}}`
	first := send("order-42", body)
	replay := send("order-42", `{"data":{"amount":1500, "order_id":42},"type":"order.created"}`)
	if first.Code != 201 || replay.Code != 200 || first.Body.String() != replay.Body.String() {
		t.Fatalf("first=%d %s replay=%d %s", first.Code, first.Body.String(), replay.Code, replay.Body.String())
	}
	for _, changed := range []string{
		`{"type":"order.created","data":{"order_id":43,"amount":1500}}`,
		`{"type":"order.updated","data":{"order_id":42,"amount":1500}}`,
	} {
		if w := send("order-42", changed); w.Code != 409 {
			t.Fatalf("conflict: %d %s", w.Code, w.Body.String())
		}
	}
	if len(store.List()) != 1 {
		t.Fatal("duplicate stored")
	}
	if w := send("another-key", body); w.Code != 201 {
		t.Fatal(w.Code)
	}
	for i := 0; i < 2; i++ {
		if w := send("", body); w.Code != 201 {
			t.Fatal(w.Code)
		}
	}
	for _, key := range []string{" ", "two words", strings.Repeat("a", 201)} {
		if w := send(key, body); w.Code != 400 {
			t.Fatalf("invalid key: %d", w.Code)
		}
	}
	if len(store.List()) != 4 {
		t.Fatal("unexpected event count")
	}
}

func TestConcurrentIdempotency(t *testing.T) {
	store := memory.NewEventStore()
	h := httpapi.NewHandler(event.NewService(store))
	var wg sync.WaitGroup
	var created atomic.Int32
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("POST", "/events", strings.NewReader(`{"type":"order.created","data":{"order_id":42}}`))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", "order-42")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code == 201 {
				created.Add(1)
			} else if w.Code != 200 {
				t.Errorf("unexpected status: %d", w.Code)
			}
		}()
	}
	wg.Wait()
	if created.Load() != 1 || len(store.List()) != 1 {
		t.Fatalf("created=%d stored=%d", created.Load(), len(store.List()))
	}
}
