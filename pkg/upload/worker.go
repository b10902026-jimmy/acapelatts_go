package upload

import (
	"fmt"
	"io"
	"log"
	"time"
	"videoUploadAndProcessing/pkg/audio_processing"
	"videoUploadAndProcessing/pkg/whisper_api"
)

const NumWorkers = 50 // 設定工作人員的數量

const InitialBackoffDuration = 500 * time.Millisecond // 初始回退時間
const MaxBackoffDuration = 16 * time.Second           // 最大回退時間

type Job struct {
	File     io.ReadCloser
	FilePath string
	APIKey   string
	Retries  int
	Done     chan bool // 用來通知工作完成
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
				if job.Retries < 2 { // 如果尚未達到最大重試次數
					job.Retries++
					backoffDuration := getBackoffDuration(job.Retries)
					log.Printf("Job failed, retrying after %v", backoffDuration)
					time.Sleep(backoffDuration) // Apply backoff delay
					w.JobQueue <- job           // 將工作重新放入佇列
				} else {
					log.Printf("Job failed after %d retries", job.Retries)
				}
			}
		}
	}()
}

// Compute the backoff duration based on retry count.
func getBackoffDuration(retryCount int) time.Duration {
	backoff := InitialBackoffDuration * time.Duration(1<<retryCount)
	if backoff > MaxBackoffDuration {
		return MaxBackoffDuration
	}
	return backoff
}

func ProcessJob(job Job) error {

	defer job.File.Close()

	log.Println("Processing vedio..") // 添加信息

	// 使用新打開的file讀取器提取音訊
	audioReader, err := audio_processing.ExtractAudioFromVideo(job.FilePath)
	if err != nil {
		log.Printf("Error extracting audio: %v", err)
		job.File.Close()
		return fmt.Errorf("error extracting audio: %v", err)
	}

	log.Println("Calling Whisper API")

	whisperAndWordTimestamps, err := whisper_api.CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
		log.Printf("Error calling Whisper API: %v", err)
		return fmt.Errorf("error calling Whisper API: %v", err)
	}

	log.Println("Generating SRT file")

	srtFilePath, err := whisper_api.CreateSRTFile(whisperAndWordTimestamps)
	if err != nil {
		log.Printf("Error creating SRT file: %v", err)
		return fmt.Errorf("error creating SRT file: %v", err)
	}

	outputPath, err := whisper_api.CreateWholeWordTimestampsFile(whisperAndWordTimestamps)
	if err != nil {

		fmt.Printf("Error creating wholeWordTimestamps file: %v\n", err)
	} else {
		fmt.Printf("Created wholeWordTimestamps file at: %s\n", outputPath)
	}

	// 讀取SRT文件
	srtSegments, err := whisper_api.ReadSRTFile(srtFilePath)
	if err != nil {
		log.Printf("Error reading SRT file: %v", err)
		return fmt.Errorf("error reading SRT file: %v", err)
	}

	log.Println("Spliting video into segments...")

	videoDuration, err := audio_processing.GetVideoDuration(job.FilePath)
	if err != nil {
		log.Printf("Failed to get video duration: %v", err)
		return fmt.Errorf("failed to get video duration: %v", err)
	}

	// Splitting video into segments and preparing for parallel processing
	allSegmentPaths, voiceSegmentPaths, err := whisper_api.SplitVideoIntoSegmentsBySRT(job.FilePath, srtSegments, videoDuration)
	if err != nil {
		log.Printf("Failed to split video into segments: %v", err)
		return fmt.Errorf("failed to split video into segments: %v", err)
	}

	log.Println("Converting audio to standard pronunciation using Acapela TTS API..") // 添加信息

	// After spliting video into many segments,create a go worker pool to handle it.
	mergedSegments, err := ProcessSegmentJobs(voiceSegmentPaths, allSegmentPaths, srtSegments)

	if err != nil {
		log.Printf("Error while processing segment workers: %v", err)
		return fmt.Errorf("error while processing segment workers: %v", err)
	}

	// 更新 allSegmentPaths
	allSegmentPaths = mergedSegments

	log.Println("Starting to merge all the video segments..")
	outputVideo, err := audio_processing.MergeAllVideoSegmentsTogether(allSegmentPaths)
	if err != nil {
		log.Printf("Failed to merge video segments into final_video: %v", err)
		return fmt.Errorf("failed to merge video segments into final_video: %v", err)
	}
	log.Printf("Successfully merged all video segments into %s", outputVideo)

	job.File.Close()

	// 工作完成，發送通知
	job.Done <- true
	return nil
}
