package audio_processing

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func execFFMPEG(args ...string) error {
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %v, output: %s", err, output)
	}
	return nil
}

/*
func cleanupFiles(files []string) {
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			log.Printf("Failed to delete file %s: %v", file, err)
		}
	}
}
*/

// GetVideoDuration 使用ffprobe來獲得影片的時長，並將時長回傳。
func GetVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0, err
	}

	durationStr := strings.TrimSpace(out.String())
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

// SplitVideoIntoSegments splits the video based on the sentence timestamps.
// It returns all segment paths and just the segment paths with voice.

func SplitVideoIntoSegments(videoPath string, sentenceTimestamps []SentenceTimestamp, videoDuration float64) ([]string, []string, error) {
	var segmentPaths []string
	var voiceSegmentPaths []string
	const outputDir = "../pkg/audio_processing/tmp/video/"

	// 檢查目錄是否存在，如果不存在則創建
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	// 預設上一個段落的結束時間為0
	lastEndTime := 0.0

	for i, ts := range sentenceTimestamps {
		// 如果當前片段的開始時間不是緊接在上一個片段的結束時間後，提取中間的片段
		if ts.StartTime > lastEndTime {
			gapOutputFile := outputDir + fmt.Sprintf("video_gap_segment%d.mp4", i)
			err := execFFMPEG("-y", "-i", videoPath, "-ss", fmt.Sprint(lastEndTime), "-t", fmt.Sprint(ts.StartTime-lastEndTime), gapOutputFile)
			if err != nil {
				return nil, nil, fmt.Errorf("error extracting gap segment: %v", err)
			}
			segmentPaths = append(segmentPaths, gapOutputFile)
		}

		// 提取有對應時間戳的片段
		outputFile := outputDir + fmt.Sprintf("video_segment%d.mp4", i)
		err := execFFMPEG("-y", "-i", videoPath, "-ss", fmt.Sprint(ts.StartTime), "-t", fmt.Sprint(ts.EndTime-ts.StartTime), outputFile)
		if err != nil {
			return nil, nil, fmt.Errorf("error splitting video into segments: %v", err)
		}

		segmentPaths = append(segmentPaths, outputFile)
		voiceSegmentPaths = append(voiceSegmentPaths, outputFile)
		// 更新上一段的結束時間
		lastEndTime = ts.EndTime
	}

	// 處理影片的最後部分，如果有的話
	if lastEndTime < videoDuration {
		gapOutputFile := outputDir + "video_end_segment.mp4"
		err := execFFMPEG("-y", "-i", videoPath, "-ss", fmt.Sprint(lastEndTime), "-t", fmt.Sprint(videoDuration-lastEndTime), gapOutputFile)
		if err != nil {
			return nil, nil, fmt.Errorf("error extracting end segment: %v", err)
		}
		segmentPaths = append(segmentPaths, gapOutputFile)
	}

	return segmentPaths, voiceSegmentPaths, nil
}

// MergeVideoAndAudio merges a video and an audio file using ffmpeg and outputs to a specified file.
func MergeVideoAndAudio(videoPath string, audioPath string, outputPath string) error {
	videoDuration, err := GetVideoDuration(videoPath)
	if err != nil {
		return fmt.Errorf("error getting video duration: %v", err)
	}

	audioDuration, err := GetVideoDuration(audioPath) // GetVideoDuration works for audio too
	if err != nil {
		return fmt.Errorf("error getting audio duration: %v", err)
	}

	setptsValue := 1.0
	if videoDuration > audioDuration {
		// If video is longer, slow down the video slightly
		setptsValue = videoDuration / audioDuration
	} else if videoDuration < audioDuration {
		// If audio is longer, speed up the video slightly
		setptsValue = videoDuration / audioDuration
	}

	// Adjust video speed and merge
	tempVideoPath := "temp_video.mp4"
	err = execFFMPEG("-y", "-i", videoPath, "-vf", fmt.Sprintf("setpts=%f*PTS", setptsValue), "-y", tempVideoPath)
	if err != nil {
		return fmt.Errorf("error adjusting video speed: %v", err)
	}

	err = execFFMPEG("-y", "-i", tempVideoPath, "-i", audioPath, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", outputPath)
	if err != nil {
		return fmt.Errorf("error merging video and audio: %v", err)
	}
	return nil
}

func GenerateSilenceAudio(duration float64, outputPath string) error {
	err := execFFMPEG("-f", "lavfi", "-y", "-i", "anullsrc", "-t", fmt.Sprint(duration), "-ar", "44100", "-ac", "2", outputPath)
	if err != nil {
		return fmt.Errorf("error generating silence audio: %v", err)
	}
	return nil
}
