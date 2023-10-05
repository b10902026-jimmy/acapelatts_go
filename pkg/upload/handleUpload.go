package upload

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	// 其他import
)

type VideoPathRequest struct {
	VideoPathToBeProcessed string `json:"video_path_to_be_processed"`
	CallbackURL            string `json:"callback_url"`
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
	unprocessedfilePath := videoPathReq.VideoPathToBeProcessed

	fileName := filepath.Base(unprocessedfilePath)

	log.Printf("FilePath: %s", unprocessedfilePath)
	log.Printf("FileName: %s", fileName)

	apiKey := os.Getenv("WHISPER_API_KEY")
	if apiKey == "" {
		http.Error(w, "WHISPER_API_KEY environment variable not set", http.StatusBadRequest)
		return
	}

	// 創建一個通道來接收工作完成的通知
	done := make(chan bool)

	// 創建一個通道來接收處理後的文件路徑
	processedFilePathChan := make(chan string)

	worker.JobQueue <- Job{
		File:                  nil,
		UnprocessedFilePath:   unprocessedfilePath,
		ProcessedFilePathChan: processedFilePathChan, // 設置通道
		APIKey:                apiKey,
		Done:                  done,
		FileName:              fileName,
	}

	// 啟動一個協程來等待工作完成並執行清理操作
	go func() {
		log.Printf("Go routine started waiting recieving data from worker channel")
		<-done
		processedFilePath := <-processedFilePathChan // 從通道中讀取處理後的文件路徑

		// 使用處理後的文件路徑作為回調中的 new_path
		payload := fmt.Sprintf(`{"status":"done", "new_path": "%s"}`, processedFilePath)
		// 打印 payload 到終端機
		log.Printf("Payload to be sent: %s", payload)

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

	w.WriteHeader(http.StatusOK)
}
