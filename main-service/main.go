package main

import (
	"log"
	"net/http"
	"os"
	"tsorgz/live-data-engine/internal/database"
	"tsorgz/live-data-engine/internal/handlers"
	"tsorgz/live-data-engine/internal/services"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	csvWatcher := services.NewCSVWatcher(os.Getenv("CSV_PATH"))
	apiClient := services.NewAPIClient(os.Getenv("EXTERNAL_API_URL"))

	http.HandleFunc("/stream", handlers.StreamHandler(db, csvWatcher, apiClient))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
