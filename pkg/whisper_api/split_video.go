package whisper_api

import (
	"fmt"
	"os"
	"videoUploadAndProcessing/pkg/audio_processing"
)

func SplitVideoIntoSegmentsByTimestamps(videoPath string, sentenceTimestamps []SentenceTimestamp, videoDuration float64) ([]string, []string, error) {
	var allSegmentPaths []string
	var voiceSegmentPaths []string
	const outputDir = "../pkg/audio_processing/tmp/video/"

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	var segmentTimes []float64
	lastEndTime := 0.0

	for i, ts := range sentenceTimestamps {
		if ts.StartTime > lastEndTime {
			gapOutputFile := outputDir + fmt.Sprintf("video_gap_segment%d.mp4", i)
			segmentTimes = append(segmentTimes, lastEndTime, ts.StartTime)
			allSegmentPaths = append(allSegmentPaths, gapOutputFile)
		}

		outputFile := outputDir + fmt.Sprintf("video_voice_segment%d.mp4", i)
		segmentTimes = append(segmentTimes, ts.StartTime, ts.EndTime)
		allSegmentPaths = append(allSegmentPaths, outputFile)
		voiceSegmentPaths = append(voiceSegmentPaths, outputFile)

		lastEndTime = ts.EndTime
	}

	if lastEndTime < videoDuration {
		endOutputFile := outputDir + "video_end_segment.mp4"
		segmentTimes = append(segmentTimes, lastEndTime, videoDuration)
		allSegmentPaths = append(allSegmentPaths, endOutputFile)
	}

	segmentTimesStr := ""
	for i, t := range segmentTimes {
		if i > 0 {
			segmentTimesStr += ","
		}
		segmentTimesStr += fmt.Sprintf("%f", t)
	}

	err := audio_processing.ExecFFMPEG("-i", videoPath, "-c", "copy", "-map", "0", "-f", "segment", "-reset_timestamps", "1", "-segment_times", segmentTimesStr, outputDir+"segment%d.mp4")
	if err != nil {
		return nil, nil, fmt.Errorf("error executing FFmpeg command: %v", err)
	}

	for i, expectedPath := range allSegmentPaths {
		actualPath := outputDir + fmt.Sprintf("segment%d.mp4", i)
		err = os.Rename(actualPath, expectedPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error renaming segment file: %v", err)
		}
	}

	return allSegmentPaths, voiceSegmentPaths, nil
}
