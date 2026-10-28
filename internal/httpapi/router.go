package httpapi

import (
	"net/http"

	"webhookhub/internal/event"
)

func NewHandler(events *event.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	h := eventHandler{events: events}
	mux.HandleFunc("POST /events", h.create)
	mux.HandleFunc("GET /events", h.list)
	mux.HandleFunc("GET /events/stats", h.stats)
	mux.HandleFunc("GET /events/{id}", h.get)
	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
