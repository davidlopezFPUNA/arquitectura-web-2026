package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
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
		fmt.Fprintf(w, "http_requests_total{service=\"media-svc\",code=\"200\"} 1\n")
	})

	http.HandleFunc("/v2/media/upload", func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req-media-gen"
		}
		w.Header().Set("X-Request-ID", reqID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message":       "Imagen subida y procesada correctamente",
			"thumbnail_url": "/v2/media/thumb_pocketbase.jpg",
			"event":         "media.uploaded",
		})
	})

	log.Printf("Media Service listo en puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
