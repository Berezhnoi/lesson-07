package main

import (
	"log"
	"net/http"
	"os"

	"homework/internal/app"
)

// @title Movie API
// @version 1.0
// @description REST API for creating, listing, replacing, and deleting movies.
// @BasePath /api/v1
func main() {
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, app.NewRouterWithSwagger()); err != nil {
		log.Fatal(err)
	}
}
