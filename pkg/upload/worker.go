package upload

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"videoUploadAndProcessing/pkg/acapela_api"
	"videoUploadAndProcessing/pkg/audio_processing"
	"videoUploadAndProcessing/pkg/whisper_api"
)

const NumWorkers = 5 // 設定工作人員的數量

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

type SegmentJob struct {
	Text       string
	VideoPath  string
	Suffix     string
	SegmentIdx int
}

type SegmentWorker struct {
	ID          int
	JobQueue    chan SegmentJob
	SegmentPath *string
	SegmentIdx  int
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

func (w SegmentWorker) Start(wg *sync.WaitGroup, errors chan<- error) {
	go func() {
		for job := range w.JobQueue {
			log.Printf("SegmentWorker %d: Starting processing for segment %d", w.ID, job.SegmentIdx)
			// Convert text to speech
			audioSegment, err := acapela_api.ConvertTextToSpeechUsingAcapela(job.Text, job.Suffix, job.SegmentIdx)
			if err != nil {
				errors <- fmt.Errorf("SegmentWorker %d: failed to convert text to speech for segment %d: %v", w.ID, job.SegmentIdx, err)
				continue
			}

			log.Printf("SegmentWorker %d: Converted text to speech for segment %d", w.ID, job.SegmentIdx)
			// Merge the voice-over with the video segment and overwrite the original segment
			var mergedSegment string
			if strings.HasSuffix(job.VideoPath, ".mp4") {
				mergedSegment = strings.TrimSuffix(job.VideoPath, ".mp4") + "_merged.mp4"
			} else {
				mergedSegment = job.VideoPath + "_merged.mp4"
			}

			err = audio_processing.MergeVideoAndAudioBySegments(job.VideoPath, audioSegment, mergedSegment, job.SegmentIdx)
			if err != nil {
				errors <- fmt.Errorf("SegmentWorker %d: failed to merge video and audio for segment %d: %v", w.ID, job.SegmentIdx, err)
				continue
			}

			log.Printf("SegmentWorker %d: Merged video and audio for segment %d", w.ID, job.SegmentIdx)

			// Store the merged segment path at the location pointed to by SegmentPath
			*w.SegmentPath = mergedSegment
			// Add a log here to trace the stored path
			log.Printf("SegmentWorker %d: Stored merged segment path for segment %d: %s", w.ID, job.SegmentIdx, *w.SegmentPath)
		}

		// Add a log here to check the final value of *w.SegmentPath
		log.Printf("SegmentWorker %d: Final stored merged segment path: %s", w.ID, *w.SegmentPath)

		// Decrement the wait group counter when done
		wg.Done()
	}()
}

func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 // not found.
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

	log.Println("Processing vedio...") // 添加信息
	log.Println("Extracting audio from uploaded video...")
	// 使用os.Open重新打開文件以供讀取
	file, err := os.Open(job.FilePath)
	if err != nil {
		log.Printf("Failed to open the file: %v", err)
		job.File.Close()
		return fmt.Errorf("failed to open the file: %v", err)
	}
	defer file.Close()

	// 使用新打開的file讀取器提取音訊
	audioReader, err := audio_processing.ExtractAudioFromVideo(file)
	if err != nil {
		log.Printf("Error extracting audio: %v", err)
		job.File.Close()
		return fmt.Errorf("error extracting audio: %v", err)
	}

	log.Println("Calling Whisper API...")

	// 使用從環境變數獲取的API key
	whisperResp, wordTimestamps, err := whisper_api.CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
		log.Printf("Error calling Whisper API: %v", err)
		return fmt.Errorf("error calling Whisper API: %v", err)
	}
	log.Println("Generating SRT file...")
	err = whisper_api.CreateSRTFile(whisperResp)
	if err != nil {
		log.Printf("Error creating SRT file: %v", err)
		return fmt.Errorf("error creating SRT file: %v", err)
	}

	log.Println("Spliting video into segments...")

	videoDuration, err := audio_processing.GetVideoDuration(job.FilePath)
	if err != nil {
		log.Printf("Failed to get video duration: %v", err)
		return fmt.Errorf("failed to get video duration: %v", err)
	}

	// Splitting video into segments and preparing for parallel processing
	allSegmentPaths, voiceSegmentPaths, err := whisper_api.SplitVideoIntoSegmentsByTimestamps(job.FilePath, wordTimestamps, videoDuration)
	if err != nil {
		log.Printf("Failed to split video into segments: %v", err)
		return fmt.Errorf("failed to split video into segments: %v", err)
	}

	log.Println("Converting audio to standard pronunciation using Acapela TTS API..") // 添加信息

	// Create a slice to hold the merged segment paths
	mergedSegments := make([]string, len(allSegmentPaths))
	copy(mergedSegments, allSegmentPaths)

	// Create segment workers
	segmentWorkers := make([]SegmentWorker, len(voiceSegmentPaths))
	for i, voiceSegment := range voiceSegmentPaths {
		idx := indexOf(voiceSegment, allSegmentPaths) // Find the index in allSegmentPaths
		segmentWorkers[i] = SegmentWorker{
			ID:          i,
			JobQueue:    make(chan SegmentJob, 1),
			SegmentPath: &mergedSegments[idx], // Pointer to the corresponding element in mergedSegments
			SegmentIdx:  idx,                  // Store the index
		}
	}

	// Create a wait group to wait for all segment workers to finish
	var wg sync.WaitGroup
	wg.Add(len(voiceSegmentPaths))

	// Create channels to collect errors
	errors := make(chan error, len(voiceSegmentPaths))

	// Start the segment workers
	for i := 0; i < len(segmentWorkers); i++ {
		segmentWorkers[i].Start(&wg, errors)
	}

	// Create and add the segment jobs
	for i := 0; i < len(voiceSegmentPaths); i++ {
		segmentJob := SegmentJob{
			Text:       wordTimestamps[i].Word, // 使用单词级时间戳的单词.Text,
			VideoPath:  voiceSegmentPaths[i],
			Suffix:     "Ryan22k_NT",
			SegmentIdx: i,
		}
		// Add the job to the worker's queue
		segmentWorkers[i].JobQueue <- segmentJob
	}

	// Close all job queues for segment workers
	for i := 0; i < len(segmentWorkers); i++ {
		close(segmentWorkers[i].JobQueue)
	}

	// Wait for all segment workers to finish
	wg.Wait()

	// Check for errors
	close(errors)
	for err := range errors {
		if err != nil {
			log.Printf("Error processing segment: %v", err)
			// Handle error as needed
		}
	}
	log.Println("Error checking complete.") // Added log

	// Update allSegmentPaths with the merged segments
	allSegmentPaths = mergedSegments

	log.Println("Starting to merge all the video segments..")
	outputVideo, err := audio_processing.MergeAllVideoSegmentsTogether(allSegmentPaths)
	if err != nil {
		log.Printf("Failed to merge video segments into final_video: %v", err)
		return fmt.Errorf("failed to merge video segments into final_video: %v", err)
	}
	log.Printf("Successfully merged all video segments into %s", outputVideo)

	// 打開outputVideo文件
	outputFile, err := os.Open(outputVideo)
	if err != nil {
		log.Printf("Failed to open the output video: %v", err)
		return fmt.Errorf("failed to open the output video: %v", err)
	}
	defer outputFile.Close()

	job.File.Close()

	// 工作完成，發送通知
	job.Done <- true
	return nil
}
