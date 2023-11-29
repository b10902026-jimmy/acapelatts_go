package video_processing

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"videoUploadAndProcessing/pkg/whisper_api"
)

func printVideoStreamInfo(videoPath string) error {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name,codec_type", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	log.Printf("Stream info for video %s: \n%s", videoPath, output)
	return nil
}

func createTemporarySRTFile(srtSegment whisper_api.SRTSegment, segmentIdx int, tempDirPrefix string) (string, error) {

	// 創建暫存路徑
	tempFileName := fmt.Sprintf("temp_%d.srt", segmentIdx)
	tempFilePath := path.Join(tempDirPrefix, "segment_srt", tempFileName)

	// 確保目錄存在
	tempDir := path.Dir(tempFilePath)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %v", tempDir, err)
	}

	// 正確使用完整路徑創建文件
	file, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %v", tempFilePath, err)
	}
	defer file.Close()

	// 將內容寫入SRT FILE
	_, err = file.WriteString(fmt.Sprintf("1\n00:00:00,000 --> 00:00:%.3f\n%s\n", srtSegment.EndTime, srtSegment.Text))
	if err != nil {
		return "", err
	}

	return tempFilePath, nil
}

func AddSubtitlesToSegment(videoPath string, srtSegment whisper_api.SRTSegment, outputPath string, segmentIdx int, tempDirPrefix string) error {
	// Reset StartTime and EndTime
	srtSegment.StartTime = 0
	srtSegment.EndTime -= srtSegment.StartTime

	// Create a temporary SRT file
	tempFileName, err := createTemporarySRTFile(srtSegment, segmentIdx, tempDirPrefix)
	if err != nil {
		return fmt.Errorf("error creating temporary SRT file for segment %d: %v", segmentIdx, err)
	}
	defer os.Remove(tempFileName)

	// Generate the FFmpeg subtitle filter string
	subtitleStr := fmt.Sprintf("subtitles='%s:si=0'", tempFileName)

	/*
		// Print video stream info
		err = printVideoStreamInfo(videoPath)
		if err != nil {
			log.Printf("Warning: Failed to print video stream info: %v", err)
		}
	*/

	// Create temporary output file
	tempOutputPath := outputPath + "_temp.mp4"

	err = execFFMPEG("-y", "-i", videoPath, "-ar", "44100", "-ac", "2", "-vf", subtitleStr, tempOutputPath)
	if err != nil {
		return fmt.Errorf("error executing FFmpeg command for segment %d: %v", segmentIdx, err)
	}

	// Replace the original video file with the new one
	err = os.Rename(tempOutputPath, outputPath)
	if err != nil {
		return fmt.Errorf("error renaming file: %v", err)
	}

	log.Printf("Subtitle added for segement %d,store at :%s", segmentIdx, outputPath)

	return nil
}
