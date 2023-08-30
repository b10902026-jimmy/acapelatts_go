package whisper_api

import (
	"fmt"
	"os"
	"strings"
	"videoUploadAndProcessing/pkg/audio_processing"
)

func SplitVideoIntoSegmentsBySRT(videoPath string, srtSegments []SRTSegment, videoDuration float64) ([]string, []string, error) {
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
	//segmentTimes = append(segmentTimes, lastEndTime)
	for i, ts := range srtSegments {
		if ts.StartTime > lastEndTime {
			gapOutputFile := outputDir + fmt.Sprintf("video_gap_segment%d.mp4", i)
			segmentTimes = append(segmentTimes, ts.StartTime)
			allSegmentPaths = append(allSegmentPaths, gapOutputFile)
		}

		outputFile := outputDir + fmt.Sprintf("video_voice_segment%d.mp4", i)
		segmentTimes = append(segmentTimes, ts.EndTime)
		allSegmentPaths = append(allSegmentPaths, outputFile)
		voiceSegmentPaths = append(voiceSegmentPaths, outputFile)

		lastEndTime = ts.EndTime
	}

	if lastEndTime < videoDuration {
		endOutputFile := outputDir + "video_end_segment.mp4"
		segmentTimes = append(segmentTimes, videoDuration)
		allSegmentPaths = append(allSegmentPaths, endOutputFile)
	}
	// 打印 allSegmentPaths 中的總片段數
	fmt.Printf("Total number of segments in allSegmentPaths: %d\n", len(allSegmentPaths))

	segmentTimesStr := ""
	for i, t := range segmentTimes {
		if i > 0 {
			segmentTimesStr += ","
		}
		segmentTimesStr += fmt.Sprintf("%f", t)
	}

	fmt.Println("Segment Times: ", segmentTimes)
	fmt.Println("Segment TimesSTR: ", segmentTimesStr)

	err := audio_processing.ExecFFMPEG("-i", videoPath,
		"-c:v", "libx264", // 使用 libx264 编码器
		"-c:a", "copy", // 复制音频流，不重新编码
		"-map", "0",
		"-f", "segment",
		"-reset_timestamps", "1",
		"-force_key_frames", segmentTimesStr, // 强制关键帧
		"-segment_times", segmentTimesStr,
		outputDir+"segment%d.mp4")

	if err != nil {
		return nil, nil, fmt.Errorf("error executing FFmpeg command: %v", err)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading directory: %v", err)
	}

	segmentCount := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".mp4") {
			segmentCount++
		}
	}

	fmt.Printf("Total number of segments splited by ffmpeg command: %d\n", segmentCount)

	for i, expectedPath := range allSegmentPaths {
		actualPath := outputDir + fmt.Sprintf("segment%d.mp4", i)
		err = os.Rename(actualPath, expectedPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error renaming segment file: %v", err)
		}
	}

	return allSegmentPaths, voiceSegmentPaths, nil
}
