package whisper_api

import (
	"fmt"
	"os"
	"videoUploadAndProcessing/pkg/audio_processing"
)

// SplitVideoIntoSegments splits the video based on the sentence timestamps.
// It returns all segment paths and just the segment paths with voice.

func SplitVideoIntoSegmentsByTimestamps(videoPath string, sentenceTimestamps []SentenceTimestamp, videoDuration float64) ([]string, []string, error) {
	var allSegmentPaths []string
	var voiceSegmentPaths []string
	const outputDir = "../pkg/audio_processing/tmp/video/"

	// 檢查目錄是否存在，如果不存在則創建
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	// Prepare segment times for FFmpeg command
	var segmentTimes []float64
	lastEndTime := 0.0
	for i, ts := range sentenceTimestamps {
		// 如果當前片段的開始時間不是緊接在上一個片段的結束時間後，提取中間的片段
		if ts.StartTime > lastEndTime {
			gapOutputFile := outputDir + fmt.Sprintf("video_gap_segment%d.mp4", i)
			segmentTimes = append(segmentTimes, ts.StartTime)
			allSegmentPaths = append(allSegmentPaths, gapOutputFile)
		}

		// Prepare for voice segments
		outputFile := outputDir + fmt.Sprintf("video_segment%d.mp4", i)
		segmentTimes = append(segmentTimes, ts.EndTime)
		allSegmentPaths = append(allSegmentPaths, outputFile)
		voiceSegmentPaths = append(voiceSegmentPaths, outputFile)

		// Update the last end time
		lastEndTime = ts.EndTime
	}

	// Handle the end part of the video
	if lastEndTime < videoDuration {
		endOutputFile := outputDir + "video_end_segment.mp4"
		allSegmentPaths = append(allSegmentPaths, endOutputFile)
	}

	// Convert segment times to string format
	segmentTimesStr := ""
	for i, t := range segmentTimes {
		if i > 0 {
			segmentTimesStr += ","
		}
		segmentTimesStr += fmt.Sprintf("%f", t)
	}

	// Execute FFmpeg command
	err := audio_processing.ExecFFMPEG("-i", videoPath, "-c", "copy", "-map", "0", "-f", "segment", "-segment_times", segmentTimesStr, outputDir+"segment%d.mp4")
	if err != nil {
		return nil, nil, fmt.Errorf("error executing FFmpeg command: %v", err)
	}

	return allSegmentPaths, voiceSegmentPaths, nil
}
