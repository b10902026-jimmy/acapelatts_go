package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const NumWorkers = 5 // 設定工作人員的數量

type Job struct {
	File     io.ReadCloser
	FileName string
	APIKey   string
	Retries  int
}

type Worker struct {
	ID       int
	JobQueue chan Job
}

func (w Worker) Start() {
	go func() {
		for job := range w.JobQueue {
			log.Printf("Worker %d processing job", w.ID)
			err := ProcessJob(job)
			if err != nil {
				if job.Retries < 5 { // 如果尚未達到最大重試次數
					job.Retries++
					w.JobQueue <- job // 將工作重新放入佇列
				} else {
					log.Printf("Job failed after %d retries", job.Retries)
				}
			}
		}
	}()
}

func ProcessJob(job Job) error {

	defer job.File.Close()
	// 提取音訊
	audioReader, err := ExtractAudioFromVideo(job.File)
	if err != nil {
		return fmt.Errorf("error extracting audio: %v", err)
	}

	// 使用從環境變數獲取的API key
	whisperResp, err := CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
		return fmt.Errorf("error calling Whisper API: %v", err)
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
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	log.Println(string(responseJSON))

	return nil
}

func handleUpload(w http.ResponseWriter, r *http.Request, worker Worker) {
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

func main() {
	jobQueue := make(chan Job, 100)
	workers := make([]Worker, NumWorkers)

	// 初始化所有的工作人員並啟動他們
	for i := 0; i < NumWorkers; i++ {
		workers[i] = Worker{
			ID:       i + 1,
			JobQueue: jobQueue,
		}
		workers[i].Start()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, workers[0]) // 此處將第一個工作人員傳遞給處理函式。實際上，由於所有工作人員共享同一個工作佇列，所以傳遞任何一個工作人員都可以。
	})

	// 使用LoggingMiddleware包裝您的路由
	http.Handle("/", LoggingMiddleware(mux))

	port := "3000"
	fmt.Printf("Starting server on port %s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
