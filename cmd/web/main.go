package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"scholarflow_web/internal/apiclient"
	"scholarflow_web/internal/web"
)

func main() {
	apiURL := envString("SCHOLARFLOW_API_URL", "http://localhost:8080")
	addr := envString("WEB_ADDR", ":8090")

	client := apiclient.New(apiclient.Config{BaseURL: apiURL})
	h := web.NewHandler(client)

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/", h.Collection)
	r.Get("/papers/{id}", h.Paper)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS()))))

	log.Printf("starting web on %s api=%s", addr, apiURL)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
