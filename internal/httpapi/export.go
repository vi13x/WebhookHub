package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"webhookhub/internal/event"
)

func (h eventHandler) export(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	for name := range query {
		if name != "type" && name != "from" && name != "to" {
			writeError(w, http.StatusBadRequest, "export supports only type, from and to")
			return
		}
	}
	from, to, err := timeBounds(query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, total := h.events.Query(event.ListOptions{
		Type: query.Get("type"), From: from, To: to, Limit: int(^uint(0) >> 1),
	})
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", `attachment; filename="events.ndjson"`)
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	for _, item := range items {
		if r.Context().Err() != nil {
			return
		}
		if err := encoder.Encode(item); err != nil {
			return
		}
	}
}
