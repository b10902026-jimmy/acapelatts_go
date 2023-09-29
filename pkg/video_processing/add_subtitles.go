package video_processing

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

func createTemporarySRTFile(srtSegment whisper_api.SRTSegment, segmentIdx int) (string, error) {
	// Create a temporary SRT file
	tempFileName := fmt.Sprintf("temp_%d.srt", segmentIdx)
	file, err := os.Create(tempFileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Write to the SRT file
	_, err = file.WriteString(fmt.Sprintf("1\n00:00:00,000 --> 00:00:%.3f\n%s\n", srtSegment.EndTime, srtSegment.Text))
	if err != nil {
		return "", err
	}

	return tempFileName, nil
}

func AddSubtitlesToSegment(videoPath string, srtSegment whisper_api.SRTSegment, outputPath string, segmentIdx int) error {
	// Reset StartTime and EndTime
	srtSegment.StartTime = 0
	srtSegment.EndTime -= srtSegment.StartTime

	// Create a temporary SRT file
	tempFileName, err := createTemporarySRTFile(srtSegment, segmentIdx)
	if err != nil {
		return fmt.Errorf("error creating temporary SRT file for segment %d: %v", segmentIdx, err)
	}
	defer os.Remove(tempFileName)

	// Generate the FFmpeg subtitle filter string
	subtitleStr := fmt.Sprintf("subtitles='%s:si=0'", tempFileName)

	// Print video stream info
	err = printVideoStreamInfo(videoPath)
	if err != nil {
		log.Printf("Warning: Failed to print video stream info: %v", err)
	}

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
