package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
		return fmt.Errorf("error extracting audio: %v", err)
	}

	// 使用從環境變數獲取的API key
	whisperResp, err := audio_processing.CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
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
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	log.Println(string(responseJSON))

	return nil
}
