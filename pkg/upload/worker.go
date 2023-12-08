package upload

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"videoUploadAndProcessing/pkg/video_processing"
	"videoUploadAndProcessing/pkg/whisper_api"
)

const NumWorkers = 50 // 設定工作人員的數量

const InitialBackoffDuration = 500 * time.Millisecond // 初始回退時間
const MaxBackoffDuration = 16 * time.Second           // 最大回退時間

type Job struct {
	File                  io.ReadCloser
	FileName              string
	UnprocessedFilePath   string
	ProcessedFilePathChan chan string // 用來傳送處理後的文件路徑
	APIKey                string
	Retries               int
	Done                  chan bool // 用來通知工作完成
}

type Worker struct {
	ID       int
	JobQueue chan Job
}

func (w Worker) Start() {
	go func() {
		for job := range w.JobQueue {
			log.Printf("Worker %d processing job", w.ID)
			err := ProcessJob(job, w.ID)

			if err != nil {
				/*if job.Retries < 2 { // 如果尚未達到最大重試次數
					job.Retries++
					backoffDuration := getBackoffDuration(job.Retries)
					log.Printf("Job failed, retrying after %v", backoffDuration)
					time.Sleep(backoffDuration)
					w.JobQueue <- job // 將工作重新放入佇列
				} else {

				}*/
				log.Printf("Job failed after %d retries", job.Retries)
			} else {
				log.Printf("worker%d job done", w.ID)
			}

		}
	}()
}

/*
// Compute the backoff duration based on retry count.
func getBackoffDuration(retryCount int) time.Duration {
	backoff := InitialBackoffDuration * time.Duration(1<<retryCount)
	if backoff > MaxBackoffDuration {
		return MaxBackoffDuration
	}
	return backoff
}*/

func createUniqueTempDir(workerID int) (string, error) {
	uniqueDir := fmt.Sprintf("tmp/worker%d", workerID)
	err := os.MkdirAll(uniqueDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create unique temp directory for worker %d: %v", workerID, err)
	}
	return uniqueDir, nil
}

func ProcessJob(job Job, workerID int) error {
	if job.File != nil {
		defer job.File.Close()
	}

	// Create a unique temporary directory for this worker
	tempDirPrefix, err := createUniqueTempDir(workerID)
	if err != nil {
		log.Fatalf("Failed to create a unique temporary directory for worker %d: %v", workerID, err)
	}
	defer os.RemoveAll(tempDirPrefix) // Schedule the cleanup of this directory when the function exits

	log.Printf("Temporary directory created at: %s", tempDirPrefix)

	// 獲取影片的metadata
	metadata, err := video_processing.GetVideoMetadata(job.UnprocessedFilePath)
	if err != nil {
		log.Printf("Failed to get video metadata: %v", err)
		return fmt.Errorf("failed to get video metadata: %v", err)
	}
	log.Printf("Video's Metadata: %+v\n", metadata)

	log.Println("Extracting aduio from video streamly")

	// 使用新打開的file讀取器提取音訊(流式)
	audioReader, err := video_processing.StreamedExtractAudioFromVideo(job.UnprocessedFilePath)
	if err != nil {
		log.Printf("Error extracting audio: %v", err)

		return fmt.Errorf("error extracting audio: %v", err)
	}

	log.Println("Calling Whisper API and wating for response")
	//呼叫STT API(whisper)
	whisperAndWordTimestamps, err := whisper_api.CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
		log.Printf("Error calling Whisper API: %v", err)
		return fmt.Errorf("error calling Whisper API: %v", err)
	}

	log.Println("Generating SRT file streamly")

	//根據STT結果創建SRT file(流式)
	srtFilePath, err := whisper_api.StreamedCreateSRTFile(whisperAndWordTimestamps, tempDirPrefix)
	if err != nil {
		log.Printf("Error creating SRT file: %v", err)
		return fmt.Errorf("error creating SRT file: %v", err)
	}

	//創建所有單詞的時間戳
	outputPath, err := whisper_api.CreateWholeWordTimestampsFile(whisperAndWordTimestamps, tempDirPrefix)
	if err != nil {
		log.Printf("Error creating wholeWordTimestamps file: %v\n", err)
	} else {
		log.Printf("Created wholeWordTimestamps file at: %s\n", outputPath)
	}

	// 讀取SRT文件
	srtSegments, err := whisper_api.ReadSRTFileFromPath(srtFilePath)
	if err != nil {
		log.Printf("Error reading SRT file: %v", err)
		return fmt.Errorf("error reading SRT file: %v", err)
	}

	//獲取影片時長
	videoDuration, err := video_processing.GetVideoDuration(job.UnprocessedFilePath)
	if err != nil {
		log.Printf("Failed to get video duration: %v", err)
		return fmt.Errorf("failed to get video duration: %v", err)
	}

	// Splitting video into segments and preparing for parallel processing
	allSegmentPaths, voiceSegmentPaths, err := video_processing.SplitVideoIntoSegmentsBySRT(job.UnprocessedFilePath, srtSegments, videoDuration, tempDirPrefix)
	if err != nil {
		log.Printf("Failed to split video into segments: %v", err)
		return fmt.Errorf("failed to split video into segments: %v", err)
	}

	log.Println("Converting audio to standard pronunciation using the Acapela TTS API and substituting the human voice with a synthesized voice...")

	// After spliting video into many segments,create a go worker pool to handle it.
	mergedSegments, err := ProcessSegmentJobs(voiceSegmentPaths, allSegmentPaths, srtSegments, tempDirPrefix)

	if err != nil {
		log.Printf("Error while processing segment workers: %v", err)
		return fmt.Errorf("error while processing segment workers: %v", err)
	}

	// 更新 allSegmentPaths
	allSegmentPaths = mergedSegments

	log.Println("Starting to merge all the video segments..")
	outputVideo, err := video_processing.MergeAllVideoSegmentsTogether(job.FileName, allSegmentPaths, tempDirPrefix)
	if err != nil {
		log.Printf("Failed to merge video segments into final_video: %v", err)
		return fmt.Errorf("failed to merge video segments into final_video: %v", err)
	} else {
		log.Printf("Successfully merged all video segments into %s", outputVideo)
	}

	// 當工作完成後

	select {
	case job.Done <- true:
		log.Println("Successfully sent true to Done channel")
	case <-time.After(time.Second * 5): // 等待5秒
		log.Println("Timeout while trying to send true to Done channel")
	}

	select {
	case job.ProcessedFilePathChan <- outputVideo:
		log.Println("Successfully sent outputVideo to ProcessedFilePathChannel")
	case <-time.After(time.Second * 5): // 等待5秒
		log.Println("Timeout while trying to send outputVideo to ProcessedFilePathChannel")
	}

	return nil
}
