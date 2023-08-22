package whisper_api

import (
	"fmt"
	"log"
	"os"
	"videoUploadAndProcessing/pkg/audio_processing"
)

// SplitVideoIntoSegments splits the video based on the sentence timestamps.
// It returns all segment paths and just the segment paths with voice.

func SplitVideoIntoSegmentsByTimestamps(videoPath string, sentenceTimestamps []SentenceTimestamp, videoDuration float64) ([]string, []string, error) {
	var allSegmentPaths []string
	var voiceSegmentPaths []string // 用於存儲僅包含語音的片段
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
			err := audio_processing.ExecFFMPEG("-y", "-i", videoPath, "-ss", fmt.Sprint(lastEndTime), "-t", fmt.Sprint(ts.StartTime-lastEndTime), gapOutputFile)
			if err != nil {
				return nil, nil, fmt.Errorf("error extracting gap segment: %v", err)
			}
			allSegmentPaths = append(allSegmentPaths, gapOutputFile)
			log.Println("Added gap segment path:", gapOutputFile)
		}

		// 提取有對應時間戳的片段
		outputFile := outputDir + fmt.Sprintf("video_segment%d.mp4", i)
		err := audio_processing.ExecFFMPEG("-y", "-i", videoPath, "-ss", fmt.Sprint(ts.StartTime), "-t", fmt.Sprint(ts.EndTime-ts.StartTime), outputFile)
		if err != nil {
			return nil, nil, fmt.Errorf("error splitting video into segments: %v", err)
		}

		allSegmentPaths = append(allSegmentPaths, outputFile)     // 添加到所有片段的列表中
		voiceSegmentPaths = append(voiceSegmentPaths, outputFile) // 添加到語音片段的列表中
		log.Println("Added voice segment path:", outputFile)
		// 更新上一段的結束時間
		lastEndTime = ts.EndTime
	}

	// 處理影片的最後部分，如果有的話
	if lastEndTime < videoDuration {
		gapOutputFile := outputDir + "video_end_segment.mp4"
		err := audio_processing.ExecFFMPEG("-y", "-i", videoPath, "-ss", fmt.Sprint(lastEndTime), "-t", fmt.Sprint(videoDuration-lastEndTime), gapOutputFile)
		if err != nil {
			return nil, nil, fmt.Errorf("error extracting end segment: %v", err)
		}
		allSegmentPaths = append(allSegmentPaths, gapOutputFile)
		log.Println("Added end segment path:", gapOutputFile)
	}

	return allSegmentPaths, voiceSegmentPaths, nil
}
