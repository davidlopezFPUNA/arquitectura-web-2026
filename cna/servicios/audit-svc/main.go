package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type AuditLog struct {
	EventID   string    `json:"event_id"`
	RequestID string    `json:"request_id"`
	Entity    string    `json:"entity"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Timestamp time.Time `json:"timestamp"`
}

type AuditStore struct {
	mu     sync.Mutex
	events map[string]AuditLog
}

var store = &AuditStore{events: make(map[string]AuditLog)}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "http_requests_total{service=\"audit-svc\",code=\"200\"} %d\n", len(store.events))
	})

	http.HandleFunc("/v2/audit/events", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req-audit-gen"
		}
		w.Header().Set("X-Request-ID", reqID)

		if r.Method == http.MethodPost {
			var event AuditLog
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			store.mu.Lock()
			// Control de Idempotencia
			if _, exists := store.events[event.EventID]; exists {
				store.mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "ignored", "reason": "duplicate event"})
				return
			}
			store.events[event.EventID] = event
			store.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
			return
		}

		store.mu.Lock()
		list := []AuditLog{}
		for _, e := range store.events {
			list = append(list, e)
		}
		store.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
	})

	log.Printf("Audit Service (PocketBase) listo en puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
