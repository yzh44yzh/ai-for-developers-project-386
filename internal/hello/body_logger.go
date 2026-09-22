package hello

import (
	"io"
	"log"
	"net/http"
)

// BodyLogger accepts POST requests and writes the request body to the log.
func BodyLogger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	log.Printf("POST %s body: %q", r.URL.Path, body)

	w.WriteHeader(http.StatusOK)
}
