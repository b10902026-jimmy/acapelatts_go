package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/xfrr/goffmpeg/transcoder"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

// 定義Whisper API的響應結構
type WhisperResponse struct {
	Text     string `json:"text"`
	Segments []struct {
		WholeWordTimestamps []struct {
			Word  string  `json:"word"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"whole_word_timestamps"`
	} `json:"segments"`
}

type WordTimestamp struct {
	Word      string  `json:"word"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

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
	audioReader, err := extractAudioFromVideo(file)
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
		whisperResp, err := callWhisperAPI("YJV4AX7FYNJKV6JNVWJJJVER4GGPUG8Y", audioReader)
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

func callWhisperAPI(apiKey string, audioReader io.Reader) (*WhisperResponse, error) {
	url := "https://transcribe.whisperapi.com"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	file, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(file, audioReader)
	if err != nil {
		return nil, err
	}

	_ = writer.WriteField("fileType", "mp3")
	_ = writer.WriteField("diarization", "false")
	_ = writer.WriteField("numSpeakers", "2")
	_ = writer.WriteField("language", "en")
	_ = writer.WriteField("task", "transcribe")

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Add("Authorization", "Bearer "+apiKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Whisper API responded with status code: %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var whisperResp WhisperResponse
	err = json.Unmarshal(body, &whisperResp)
	if err != nil {
		log.Printf("Error unmarshaling Whisper API response: %v", err)
		return nil, err
	}
	log.Printf("Whisper API response unmarshaled successfully")
	log.Printf("Whisper API response text: %s", whisperResp.Text)
	return &whisperResp, nil
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lrw, r)
		duration := time.Since(start)
		log.Printf("%s %s %d %s", r.Method, r.RequestURI, lrw.status, duration)
	})
}

func extractAudioFromVideo(inputFile io.Reader) (io.Reader, error) {
	inputFilePath := "input.mp4"
	outputFilePath := "output.mp3"

	// 將上傳的文件保存到臨時文件中
	inputFileBytes, err := ioutil.ReadAll(inputFile)
	if err != nil {
		return nil, fmt.Errorf("Error reading input file: %v", err)
	}

	err = ioutil.WriteFile(inputFilePath, inputFileBytes, 0644)
	if err != nil {
		return nil, fmt.Errorf("Error writing input file to disk: %v", err)
	}

	// 初始化轉碼器
	trans := new(transcoder.Transcoder)

	// 配置轉碼器
	err = trans.Initialize(inputFilePath, outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("Error initializing transcoder: %v", err)
	}

	// 開始轉碼
	done := trans.Run(false)

	// 等待轉碼完成
	err = <-done
	if err != nil {
		return nil, fmt.Errorf("Error transcoding: %v", err)
	}

	// 讀取輸出音訊文件
	outputFileBytes, err := ioutil.ReadFile(outputFilePath)
	if err != nil {
		log.Printf("Error reading output file: %v", err)
		return nil, fmt.Errorf("Error reading output file: %v", err)
	}
	log.Printf("Audio extracted successfully")

	// 刪除臨時文件
	err = os.Remove(inputFilePath)
	if err != nil {
		log.Printf("Error deleting input file: %v", err)
	}

	err = os.Remove(outputFilePath)
	if err != nil {
		log.Printf("Error deleting output file: %v", err)
	}

	return bytes.NewReader(outputFileBytes), nil
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", handleUpload)

	// 使用loggingMiddleware包裝您的路由
	http.Handle("/", loggingMiddleware(mux))

	port := "3000"
	fmt.Printf("Starting server on port %s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
