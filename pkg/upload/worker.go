package upload

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"time"
	"videoUploadAndProcessing/pkg/audio_processing"
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

	// 使用從環境變數獲取的API key
	whisperResp, err := audio_processing.CallWhisperAPI(job.APIKey, audioReader)
	if err != nil {
		log.Printf("Error calling Whisper API: %v", err)
		return fmt.Errorf("error calling Whisper API: %v", err)
	}

	sentenceTimestamps := []audio_processing.SentenceTimestamp{}
	for _, segment := range whisperResp.Segments {
		sentenceTimestamp := audio_processing.SentenceTimestamp{
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

	videoSegmentPaths, voiceSegmentPaths, err := audio_processing.SplitVideoIntoSegments(job.FilePath, sentenceTimestamps, videoDuration)
	if err != nil {
		log.Printf("Failed to split video into segments: %v", err)
		return fmt.Errorf("failed to split video into segments: %v", err)
	}

	for i, segment := range voiceSegmentPaths {
		audioSegment, err := audio_processing.ConvertTextToSpeechUsingAcapela(whisperResp.Segments[i].Text, "Ryan22k_NT", i)
		if err != nil {
			log.Printf("Failed to convert text to speech for segment: %v", err)
			return fmt.Errorf("failed to convert text to speech for segment %d: %v", i, err)
		}

		// Merge the voice-over with the video segment and overwrite the original segment
		err = audio_processing.MergeVideoAndAudio(segment, audioSegment, segment) // Overwrite the segment
		if err != nil {
			log.Printf("Failed to merge video and audio for segment: %v", err)
			return fmt.Errorf("failed to merge video and audio for segment %d: %v", i, err)
		}
	}

	listFile := "filelist.txt"
	f, err := os.Create(listFile)
	if err != nil {
		log.Printf("Failed to create list file: %v", err)
		return fmt.Errorf("failed to create list file: %v", err)
	}
	defer f.Close()

	for _, segmentPath := range videoSegmentPaths {
		_, err = f.WriteString(fmt.Sprintf("file '%s'\n", segmentPath))
		if err != nil {
			log.Printf("Failed to write segment path to list file: %v", err)
			return fmt.Errorf("failed to write segment path to list file: %v", err)
		}
	}

	finalVideoDir := "../pkg/audio_processing/tmp/final_video"
	outputVideo := path.Join(finalVideoDir, "final_output.mp4")
	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputVideo)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to merge video segments: %v", err)
	}

	err = os.Remove(listFile)
	if err != nil {
		log.Printf("warning: failed to remove list file: %v", err)
	}

	job.File.Close()
	return nil
}
