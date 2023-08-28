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
	tempFilePath := filepath.Join("..", "pkg", "audio_processing", "tmp", "uploaded", uniqueFileName)

	// Ensure the directory exists
	dir := filepath.Dir(tempFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("error creating directories: %v", err), http.StatusInternalServerError)
		return
	}

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

	// 創建一個通道來接收工作完成的通知
	done := make(chan bool)

	worker.JobQueue <- Job{File: tempFile, FilePath: tempFilePath, APIKey: apiKey, Done: done}

	// 啟動一個協程來等待工作完成並執行清理操作
	go func() {
		<-done                  // 等待工作完成的通知
		os.Remove(tempFilePath) // 刪除原始影片文件
	}()

	w.WriteHeader(http.StatusOK)
}
