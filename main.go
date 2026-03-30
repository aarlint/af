package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aarlint/af/internal/db"
	"github.com/aarlint/af/internal/handlers"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "af.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := db.Init(database); err != nil {
		log.Fatalf("init db: %v", err)
	}

	h := &handlers.Handler{DB: database}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/actions", h.Actions)
	mux.HandleFunc("GET /api/scores", h.Scores)
	mux.HandleFunc("GET /health", handlers.Health)
	mux.Handle("/", http.FileServer(http.Dir("public")))

	log.Printf("af listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
