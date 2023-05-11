package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("video_file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 提取音訊
	audioReader, err := ExtractAudioFromVideo(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error extracting audio: %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resultChan := make(chan string, 1)

	// 將處理過程放在單獨的goroutine中
	go func() {
		// 調用Whisper API並獲取響應
		whisperResp, err := CallWhisperAPI("VSAC4PSGB3WXVGN7LRH3ANE5X1DQZVUQ", audioReader)
		if err != nil {
			resultChan <- fmt.Sprintf("Error calling Whisper API: %v", err)
			return
		}

		// 將單詞時間戳記以JSON格式返回給前端
		wordTimestamps := []WordTimestamp{}
		for _, segment := range whisperResp.Segments {
			for _, wholeWordTimestamp := range segment.WholeWordTimestamps {
				wordTimestamp := WordTimestamp{
					Word:      wholeWordTimestamp.Word,
					StartTime: wholeWordTimestamp.Start,
					EndTime:   wholeWordTimestamp.End,
				}

				wordTimestamps = append(wordTimestamps, wordTimestamp)
			}
		}

		responseJSON, err := json.Marshal(map[string]interface{}{
			"text":            whisperResp.Text,
			"word_timestamps": wordTimestamps,
		})
		if err != nil {
			resultChan <- fmt.Sprintf("Error marshaling JSON: %v", err)
			return
		}

		resultChan <- string(responseJSON)
	}()

	// 等待goroutine完成並返回結果
	select {
	case res := <-resultChan:
		if res[:5] == "Error" {
			http.Error(w, res, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, res)
	case <-ctx.Done():
		http.Error(w, "Request timeout", http.StatusRequestTimeout)
		return
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", handleUpload)

	// 使用LoggingMiddleware包裝您的路由
	http.Handle("/", LoggingMiddleware(mux))

	port := "3000"
	fmt.Printf("Starting server on port %s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
