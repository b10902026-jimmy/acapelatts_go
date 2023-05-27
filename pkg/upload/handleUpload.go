package upload

import (
	"fmt"
	"net/http"
	"os"
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

	// 從系統環境變數中獲取API key
	apiKey := os.Getenv("WHISPER_API_KEY")
	if apiKey == "" {
		http.Error(w, "WHISPER_API_KEY environment variable not set", http.StatusBadRequest)
		return
	}

	// 將工作加入佇列
	worker.JobQueue <- Job{File: file, APIKey: apiKey}

	w.WriteHeader(http.StatusOK)
}
