package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Record struct {
	ID        string `json:"id"`
	Nombre    string `json:"nombre"`
	Email     string `json:"email"`
	Created   string `json:"created"`
}

type SearchService struct {
	mu    sync.RWMutex
	index map[string]Record
}

var (
	reqCount    int
	serviceName = "search-svc"
)

func logJSON(level, msg, reqID string) {
	entry := map[string]string{
		"timestamp":  time.Now().Format(time.RFC3339),
		"level":      level,
		"service":    serviceName,
		"request_id": reqID,
		"message":    msg,
	}
	data, _ := json.Marshal(entry)
	fmt.Println(string(data))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	svc := &SearchService{index: make(map[string]Record)}

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
		fmt.Fprintf(w, "# HELP http_requests_total Total number of HTTP requests\n")
		fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
		fmt.Fprintf(w, "http_requests_total{service=\"search-svc\",code=\"200\"} %d\n", reqCount)
		fmt.Fprintf(w, "# HELP http_request_duration_seconds Latency histogram\n")
		fmt.Fprintf(w, "http_request_duration_seconds_bucket{le=\"0.1\"} %d\n", reqCount)
	})

	http.HandleFunc("/v2/search", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req-auto-generated"
		}
		w.Header().Set("X-Request-ID", reqID)
		reqCount++
		logJSON("INFO", "Ejecutando consulta de búsqueda en PocketBase index", reqID)

		query := strings.ToLower(r.URL.Query().Get("q"))
		svc.mu.RLock()
		defer svc.mu.RUnlock()

		results := []Record{}
		for _, rec := range svc.index {
			if strings.Contains(strings.ToLower(rec.Nombre), query) || strings.Contains(strings.ToLower(rec.Email), query) {
				results = append(results, rec)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"total":   len(results),
			"results": results,
		})
	})

	logJSON("INFO", "Search Service (PocketBase) escuchando en puerto "+port, "sys")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
