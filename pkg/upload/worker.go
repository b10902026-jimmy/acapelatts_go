package upload

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
	log.Println("Extracting audio from uploaded video..")
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

	log.Println("Calling Whisper API") // 添加信息

	// 使用從環境變數獲取的API key
	whisperResp, err := whisper_api.CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
		log.Printf("Error calling Whisper API: %v", err)
		return fmt.Errorf("error calling Whisper API: %v", err)
	}

	log.Println("Spliting video into segments...") // 添加信息

	sentenceTimestamps := []whisper_api.SentenceTimestamp{}
	for _, segment := range whisperResp.Segments {
		sentenceTimestamp := whisper_api.SentenceTimestamp{
			Sentence:  segment.Text,
			StartTime: segment.Start,
			EndTime:   segment.End,
		}
		sentenceTimestamps = append(sentenceTimestamps, sentenceTimestamp)
	}

	videoDuration, err := audio_processing.GetVideoDuration(job.FilePath)
	if err != nil {
		log.Printf("Failed to get video duration: %v", err)
		return fmt.Errorf("failed to get video duration: %v", err)
	}

	videoSegmentPaths, voiceSegmentPaths, err := whisper_api.SplitVideoIntoSegmentsByTimestamps(job.FilePath, sentenceTimestamps, videoDuration)
	if err != nil {
		log.Printf("Failed to split video into segments: %v", err)
		return fmt.Errorf("failed to split video into segments: %v", err)
	}

	log.Println("Converting audio to standard pronunciation using Acapela TTS API..") // 添加信息

	for i, segment := range voiceSegmentPaths {
		audioSegment, err := acapela_api.ConvertTextToSpeechUsingAcapela(whisperResp.Segments[i].Text, "Ryan22k_NT", i)
		if err != nil {
			log.Printf("Failed to convert text to speech for segment: %v", err)
			return fmt.Errorf("failed to convert text to speech for segment %d: %v", i, err)
		}

		// Merge the voice-over with the video segment and overwrite the original segment
		var mergedSegment string
		if strings.HasSuffix(segment, ".mp4") {
			mergedSegment = strings.TrimSuffix(segment, ".mp4") + "_merged.mp4"
		} else {
			mergedSegment = segment + "_merged.mp4"
		}

		err = audio_processing.MergeVideoAndAudioBySegments(segment, audioSegment, mergedSegment)
		if err != nil {
			log.Printf("Failed to merge video and audio for segment: %v", err)
			return fmt.Errorf("failed to merge video and audio for segment %d: %v", i, err)
		}
		// Find the index of the original segment in the videoSegmentPaths
		originalSegmentIndex := -1
		for j, videoSegment := range videoSegmentPaths {
			if videoSegment == segment {
				originalSegmentIndex = j
				break
			}
		}

		if originalSegmentIndex != -1 {
			// Replace the original segment path with the merged segment path
			videoSegmentPaths[originalSegmentIndex] = mergedSegment
		} else {
			log.Printf("Warning: original segment not found in videoSegmentPaths: %s", segment)
		}
	}

	log.Println("Merging all the video segments..") // 添加信息

	outputVideo, err := audio_processing.MergeAllVideoSegmentsTogether(videoSegmentPaths)
	if err != nil {
		log.Printf("Failed to merge video segments into final_video: %v", err)
		return fmt.Errorf("failed to merge video segments into final_video: %v", err)
	}

	// 打開outputVideo文件
	outputFile, err := os.Open(outputVideo)
	if err != nil {
		log.Printf("Failed to open the output video: %v", err)
		return fmt.Errorf("failed to open the output video: %v", err)
	}
	defer outputFile.Close()

	// 讀取前512個字節
	buffer := make([]byte, 512)
	_, err = outputFile.Read(buffer)
	if err != nil && err != io.EOF {
		log.Printf("Failed to read the output video: %v", err)
		return fmt.Errorf("failed to read the output video: %v", err)
	}

	// 使用http.DetectContentType檢測MIME類型
	contentType := http.DetectContentType(buffer)
	if contentType != "video/mp4" {
		log.Printf("Invalid content type detected for output video: %s", contentType)
		return fmt.Errorf("invalid content type detected for output video: %s", contentType)
	}
	log.Println("Output video is in MP4 format and appears to be intact.")

	log.Println("Video processing is complete, and the audio has been replaced with Acapela's TTS output.") // 添加信息

	job.File.Close()
	return nil
}
