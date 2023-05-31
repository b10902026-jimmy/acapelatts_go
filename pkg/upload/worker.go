package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"videoUploadAndProcessing/pkg/audio_processing"
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
	audioReader, err := audio_processing.ExtractAudioFromVideo(job.File)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("error extracting audio: %v", err)
	}

	// 使用從環境變數獲取的API key
	whisperResp, err := audio_processing.CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("error calling Whisper API: %v", err)
	}

	// 將單詞時間戳記以JSON格式返回給前端
	wordTimestamps := []audio_processing.WordTimestamp{}
	for _, segment := range whisperResp.Segments {
		for _, wholeWordTimestamp := range segment.WholeWordTimestamps {
			wordTimestamp := audio_processing.WordTimestamp{
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
		log.Println(err)
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	log.Println(string(responseJSON))
	// 為 Acapela API 呼叫製作一個文本與語音變數
	text := whisperResp.Text
	voice := "Ryan22k_NT" // 這裡請用你想用的語音

	// 呼叫 Acapela API 並取得音訊
	acapelaResp, err := audio_processing.CallAcapelaAPI(text, voice)
	if err != nil {
		return fmt.Errorf("error calling Acapela API: %v", err)
	}

	// 檢查 Acapela返回格式是否為mp3
	contentType := http.DetectContentType(acapelaResp.Content)
	if contentType != "audio/mpeg" {
		fmt.Println("The content is not in MP3 format")
		return fmt.Errorf("error: the content is not in MP3 format")
	} else {
		fmt.Println("The content of Acapela's response is in MP3 format")
	}

	// 保存音频文件
	err = audio_processing.SaveAudioToFile(acapelaResp.Content, "acapela_response_test.mp3")
	if err != nil {
		return fmt.Errorf("error saving audio to file: %v", err)
	}

	return nil
}
