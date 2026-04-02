package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/mnorrsken/augur/pkg/extender"
)

func main() {
	addr := os.Getenv("AUGUR_LISTEN_ADDR")
	if addr == "" {
		addr = ":8888"
	}

	handler := extender.NewHandler()

	http.HandleFunc("/filter", wrapJSON(handler.FilterHandler))
	http.HandleFunc("/prioritize", wrapJSON(handler.PrioritizeHandler))
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("augur-extender listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// wrapJSON wraps an extender handler so that the request is decoded from JSON
// and the response is encoded back as JSON.
func wrapJSON(fn func(r *http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
