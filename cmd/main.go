package main

import (
	"fmt"
	"net/http"
	"videoUploadAndProcessing/pkg/log"
	"videoUploadAndProcessing/pkg/upload"
)

func main() {
	jobQueue := make(chan upload.Job, 100)
	workers := make([]upload.Worker, upload.NumWorkers)

	// 初始化所有的工作人員並啟動他們
	for i := 0; i < upload.NumWorkers; i++ {
		workers[i] = upload.Worker{
			ID:       i + 1,
			JobQueue: jobQueue,
		}
		workers[i].Start()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		upload.HandleUpload(w, r, workers[0]) // 此處將第一個工作人員傳遞給處理函式。實際上，由於所有工作人員共享同一個工作佇列，所以傳遞任何一個工作人員都可以。
	})

	// 使用LoggingMiddleware包裝您的路由
	http.Handle("/", log.LoggingMiddleware(mux))

	port := "3000"
	fmt.Printf("Starting server on port %s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
