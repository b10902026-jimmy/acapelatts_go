package upload

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// @Schema
// description: Video path request payload
// required: true
type VideoPathRequest struct {
	// @Field example:/path/to/video description:"Path to the video to be processed"
	VideoPathToBeProcessed string `json:"video_path_to_be_processed"`
	// @Field example:http://callback.url description:"Callback URL for job status"
	CallbackURL string `json:"callback_url"`
}

// @Summary Upload a new video for processing
// @Description Uploads a video and triggers its processing.
// @Tags video
// @Accept json
// @Produce json
// @Param request body VideoPathRequest true "Video upload payload"
// @Success 200 {object} VideoPathRequest "Successfully uploaded"
// @Failure 400 {object} string "Bad Request"
// @Failure 405 {object} string "Method Not Allowed"
// @Router /new_uploaded [post]

// HandleUpload is the HTTP handler for video uploads
func HandleUpload(w http.ResponseWriter, r *http.Request, worker Worker) {
	// Check if the HTTP method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the JSON payload from the incoming request
	decoder := json.NewDecoder(r.Body)
	var videoPathReq VideoPathRequest
	err := decoder.Decode(&videoPathReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("error decoding JSON: %v", err), http.StatusBadRequest)
		return
	}

	prefix := os.Getenv("UNPROCESSED_VIDEO_PATH")

	// Check if the path starts with the specified prefix
	if !strings.HasPrefix(videoPathReq.VideoPathToBeProcessed, prefix) {
		http.Error(w, "Invalid video path prefix", http.StatusBadRequest)
		return
	}

	// Extract the path of the unprocessed video file from the request payload
	unprocessedfilePath := videoPathReq.VideoPathToBeProcessed

	// Extract the file name from the unprocessed file path
	fileName := filepath.Base(unprocessedfilePath)

	// Log details for debugging
	log.Printf("FilePath: %s", unprocessedfilePath)
	log.Printf("FileName: %s", fileName)

	// Retrieve API key from environment variables
	apiKey := os.Getenv("WHISPER_API_KEY")
	if apiKey == "" {
		http.Error(w, "WHISPER_API_KEY environment variable not set", http.StatusBadRequest)
		return
	}

	// Create a channel to receive a completion signal from the worker
	done := make(chan bool)

	// Create a channel to receive the processed file path from the worker
	processedFilePathChan := make(chan string)

	// Send a job to the worker's job queue
	worker.JobQueue <- Job{
		File:                  nil,
		FileName:              fileName,
		UnprocessedFilePath:   unprocessedfilePath,
		ProcessedFilePathChan: processedFilePathChan,
		APIKey:                apiKey,
		Done:                  done,
	}

	// Launch a goroutine to wait for job completion and execute cleanup operations
	go func() {
		log.Printf("Go routine started waiting for data from worker channel")
		<-done                                       // Wait until the worker signals that the job is done
		processedFilePath := <-processedFilePathChan // Get the processed file path from the channel

		// Build and log the payload for the callback
		payload := fmt.Sprintf(`{"status":"done", "processed_video_path": "%s"}`, processedFilePath)
		log.Printf("Payload to be sent: %s", payload)

		// Send the payload to the callback URL
		resp, err := http.Post(videoPathReq.CallbackURL, "application/json", strings.NewReader(payload))
		if err != nil {
			log.Printf("Failed to send callback: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("Received non-OK status code from callback: %d", resp.StatusCode)
		}
	}()

	// Send an HTTP OK status to indicate successful initiation
	w.WriteHeader(http.StatusOK)

	// Add a response message
	responseMessage := "Processing video at the specified path, please wait for callback."
	_, err = w.Write([]byte(responseMessage))
	if err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
