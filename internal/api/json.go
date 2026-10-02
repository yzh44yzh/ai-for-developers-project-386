package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{msg})
}

// decodeJSON decodes the request body into v, rejecting unknown fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// parseUUID validates s as a UUID, writing a 400 response on failure.
func parseUUID(w http.ResponseWriter, name, s string) bool {
	if _, err := uuid.Parse(s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return false
	}
	return true
}
