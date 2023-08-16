package upload

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func HandleUpload(w http.ResponseWriter, r *http.Request, worker Worker) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("video_file")
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting file: %v", err), http.StatusBadRequest)
		return
	}

	uniqueFileName := fmt.Sprintf("video_%d.mp4", time.Now().UnixNano())
	tempFilePath := filepath.Join("..", "pkg", "tmp", "video", uniqueFileName)

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		file.Close() // Close the file from the request to release resources
		http.Error(w, fmt.Sprintf("error creating temp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("error saving file: %v", err), http.StatusInternalServerError)
		return
	}

	apiKey := os.Getenv("WHISPER_API_KEY")
	if apiKey == "" {
		http.Error(w, "WHISPER_API_KEY environment variable not set", http.StatusBadRequest)
		return
	}

	worker.JobQueue <- Job{File: tempFile, FilePath: tempFilePath, APIKey: apiKey}

	w.WriteHeader(http.StatusOK)
}
