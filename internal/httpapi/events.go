package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"webhookhub/internal/event"
)

type eventHandler struct {
	events *event.Service
}

func (h eventHandler) create(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	var input struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&input)
	if err == nil {
		if nextErr := decoder.Decode(new(any)); nextErr != io.EOF {
			err = nextErr
			if err == nil {
				err = errors.New("multiple JSON values")
			}
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "body must not exceed 1 MiB")
		} else {
			writeError(w, http.StatusBadRequest, "expected one JSON object with type and data")
		}
		return
	}
	e, err := h.events.Create(input.Type, input.Data)
	if errors.Is(err, event.ErrInvalid) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create event")
		return
	}
	w.Header().Set("Location", "/events/"+e.ID)
	writeJSON(w, http.StatusCreated, e)
}

func (h eventHandler) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.events.List())
}

func (h eventHandler) get(w http.ResponseWriter, r *http.Request) {
	e, err := h.events.Get(r.PathValue("id"))
	if errors.Is(err, event.ErrNotFound) {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not get event")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
