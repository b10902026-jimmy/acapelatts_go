package video_processing

import (
	"fmt"
	"log"
	"os"
	"strings"
	"videoUploadAndProcessing/pkg/whisper_api"
)

type VideoSegment struct {
	Path     string
	Duration float64
}

func SplitVideoIntoSegmentsBySRT(videoPath string, srtSegments []whisper_api.SRTSegment, videoDuration float64) ([]string, []string, error) {
	var allSegmentPaths []string
	var voiceSegmentPaths []string
	const outputDir = "../pkg/video_processing/tmp/video/"

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	var segmentTimes []float64

	var gapAndEndSegmentInfo []VideoSegment
	lastEndTime := 0.0
	//segmentTimes = append(segmentTimes, lastEndTime)
	for i, ts := range srtSegments {
		// For gaps
		if ts.StartTime > lastEndTime {
			gapDuration := ts.StartTime - lastEndTime
			gapOutputFile := outputDir + fmt.Sprintf("video_gap_segment%d.mp4", i)
			segmentTimes = append(segmentTimes, ts.StartTime)
			allSegmentPaths = append(allSegmentPaths, gapOutputFile)
			gapAndEndSegmentInfo = append(gapAndEndSegmentInfo, VideoSegment{Path: gapOutputFile, Duration: gapDuration})
		}

		// For voice segments
		outputFile := outputDir + fmt.Sprintf("video_voice_segment%d.mp4", i)
		segmentTimes = append(segmentTimes, ts.EndTime)
		allSegmentPaths = append(allSegmentPaths, outputFile)
		voiceSegmentPaths = append(voiceSegmentPaths, outputFile)

		lastEndTime = ts.EndTime
	}

	// For end segments
	if lastEndTime < videoDuration {
		endDuration := videoDuration - lastEndTime
		endOutputFile := outputDir + "video_end_segment.mp4"
		segmentTimes = append(segmentTimes, videoDuration)
		allSegmentPaths = append(allSegmentPaths, endOutputFile)
		gapAndEndSegmentInfo = append(gapAndEndSegmentInfo, VideoSegment{Path: endOutputFile, Duration: endDuration})
	}
	// 打印 allSegmentPaths 中的總片段數
	log.Printf("Total number of segments in allSegmentPaths: %d\n", len(allSegmentPaths))

	segmentTimesStr := ""
	for i, t := range segmentTimes {
		if i > 0 {
			segmentTimesStr += ","
		}
		segmentTimesStr += fmt.Sprintf("%f", t)
	}

	log.Println("Segment Times: ", segmentTimes)
	log.Println("Segment TimesSTR: ", segmentTimesStr)

	err := execFFMPEG("-i", videoPath,
		"-c:v", "libx264",
		"-c:a", "copy",
		"-map", "0",
		"-f", "segment",
		"-reset_timestamps", "1",
		"-force_key_frames", segmentTimesStr, // 強制賦予關鍵幀
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

	log.Printf("Total number of segments splited by ffmpeg command: %d\n", segmentCount)

	for i, expectedPath := range allSegmentPaths {
		actualPath := outputDir + fmt.Sprintf("segment%d.mp4", i)
		err = os.Rename(actualPath, expectedPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error renaming segment file: %v", err)
		}
	}

	// 找出gap和end片段
	var nonVoiceSegmentPaths []string
	var nonVoiceDurations []float64
	var gapAndEndIndex = 0 // 用來追蹤 gapAndEndSegmentInfo 的索引
	for _, path := range allSegmentPaths {
		if !contains(voiceSegmentPaths, path) {
			// 檢查索引是否在 gapAndEndSegmentInfo 範圍內
			if gapAndEndIndex >= len(gapAndEndSegmentInfo) {
				return nil, nil, fmt.Errorf("index out of range for gapAndEndSegmentInfo")
			}
			nonVoiceSegmentPaths = append(nonVoiceSegmentPaths, path)
			nonVoiceDurations = append(nonVoiceDurations, gapAndEndSegmentInfo[gapAndEndIndex].Duration)
			gapAndEndIndex++ // 增加 gapAndEndSegmentInfo 的索引
		}
	}

	// 生成靜音並覆蓋原文件
	for i, path := range nonVoiceSegmentPaths {
		// 生成靜音音軌
		tempAudioPath := outputDir + "temp_audio.aac"
		durationStr := fmt.Sprintf("%f", nonVoiceDurations[i])
		err := execFFMPEG("-y", "-f", "lavfi", "-t", durationStr, "-i", "anullsrc=r=44100:cl=stereo", tempAudioPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating silent audio: %v", err)
		}

		// 合併靜音音軌與原片段視頻
		tempVideoPath := outputDir + "temp_video.mp4"
		err = execFFMPEG("-i", path, "-i", tempAudioPath, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", "-map", "0:v", "-map", "1:a", tempVideoPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error merging video and audio: %v", err)
		}

		// 覆蓋原先的路徑
		err = os.Rename(tempVideoPath, path)
		if err != nil {
			return nil, nil, fmt.Errorf("error renaming segment file: %v", err)
		}
	}

	return allSegmentPaths, voiceSegmentPaths, nil
}

// contains 函數用於檢查一個字符串切片是否包含某個字符串
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
