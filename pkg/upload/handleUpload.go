package upload

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	// 其他import
)

type VideoPathRequest struct {
	VideoPathToBeProcessed string `json:"video_path_to_be_processed"`
}

func HandleUpload(w http.ResponseWriter, r *http.Request, worker Worker) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析JSON請求
	decoder := json.NewDecoder(r.Body)
	var videoPathReq VideoPathRequest
	err := decoder.Decode(&videoPathReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("error decoding JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 現在，videoPathReq.VideoPathToBeProcessed 包含影片的路徑
	FilePath := videoPathReq.VideoPathToBeProcessed

	log.Printf("FilePath: %s", FilePath)

	apiKey := os.Getenv("WHISPER_API_KEY")
	if apiKey == "" {
		http.Error(w, "WHISPER_API_KEY environment variable not set", http.StatusBadRequest)
		return
	}

	// 創建一個通道來接收工作完成的通知
	done := make(chan bool)

	worker.JobQueue <- Job{File: nil, FilePath: FilePath, APIKey: apiKey, Done: done} // File為nil，因為現在使用路徑

	// 啟動一個協程來等待工作完成並執行清理操作
	go func() {
		<-done
		// 其他清理操作
	}()

	w.WriteHeader(http.StatusOK)
}
