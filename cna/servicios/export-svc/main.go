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

type Job struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"` // queued, running, done, failed
	Format    string    `json:"format"`
	Download  string    `json:"download_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type JobManager struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

var manager = &JobManager{jobs: make(map[string]*Job)}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
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
		fmt.Fprintf(w, "http_requests_total{service=\"export-svc\",code=\"200\"} %d\n", len(manager.jobs))
	})

	http.HandleFunc("/v2/export", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req-export-gen"
		}
		w.Header().Set("X-Request-ID", reqID)

		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())
		job := &Job{
			JobID:     jobID,
			Status:    "queued",
			Format:    "csv",
			CreatedAt: time.Now(),
		}

		manager.mu.Lock()
		manager.jobs[jobID] = job
		manager.mu.Unlock()

		go func(j *Job) {
			time.Sleep(2 * time.Second)
			j.Status = "running"
			time.Sleep(3 * time.Second)
			j.Status = "done"
			j.Download = fmt.Sprintf("http://192.168.1.25/v2/export/files/%s.csv", j.JobID)
		}(job)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(job)
	})

	http.HandleFunc("/v2/export/status", func(w http.ResponseWriter, r *http.Request) {
		jobID := r.URL.Query().Get("job_id")
		manager.mu.Lock()
		job, exists := manager.jobs[jobID]
		manager.mu.Unlock()

		if !exists {
			http.Error(w, "Job no encontrado", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	})

	log.Printf("Export Service listo en puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
