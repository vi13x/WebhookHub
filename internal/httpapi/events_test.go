package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
