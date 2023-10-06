package main

import (
	"fmt"
	"net/http"
	"videoUploadAndProcessing/pkg/log"
	"videoUploadAndProcessing/pkg/upload"
)

func main() {
	// Create a new job queue with 100 slots.
	jobQueue := make(chan upload.Job, 100)

	// Initialize a slice of workers based on the defined number of workers in the upload package.
	workers := make([]upload.Worker, upload.NumWorkers)

	// Initialize and start all the workers.
	for i := 0; i < upload.NumWorkers; i++ {
		workers[i] = upload.Worker{
			ID:       i + 1,    // Assign a unique ID to each worker starting from 1.
			JobQueue: jobQueue, // All workers share the same job queue.
		}
		workers[i].Start() // Start the worker.
	}

	// Create a new HTTP ServeMux.
	mux := http.NewServeMux()

	// Register a new route that handles video uploads.
	mux.HandleFunc("/new_uploaded", func(w http.ResponseWriter, r *http.Request) {
		// Use the first worker to handle the upload.
		// In reality, any worker could handle this since they all share the same job queue.
		upload.HandleUpload(w, r, workers[0])
	})

	// Wrap the mux with logging middleware.
	http.Handle("/", log.LoggingMiddleware(mux))

	// Define the port for the server.
	port := "30016"
	fmt.Printf("Starting server on port %s\n", port)

	// Start the HTTP server.
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
