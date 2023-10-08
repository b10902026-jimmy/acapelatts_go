package video_processing

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

// MergeVideoAndAudio merges a video and an audio file using ffmpeg and outputs to a specified file.
func MergeVideoAndAudioBySegments(videoPath string, audioPath string, outputPath string, segmentIdx int) error {
	tempAudioPath := fmt.Sprintf("temp_audio_segment_%d.mp3", segmentIdx) // Unique name

	// Defer the removal of the temporary audio file
	defer func() {
		err := os.Remove(tempAudioPath)
		if err != nil {
			log.Printf("Warning: failed to remove temporary audio file: %v", err)
		}
	}()

	videoDuration, err := GetVideoDuration(videoPath)
	if err != nil {
		return fmt.Errorf("error getting video duration: %v", err)
	}

	audioDuration, err := GetVideoDuration(audioPath) // GetVideoDuration works for audio too
	if err != nil {
		return fmt.Errorf("error getting audio duration: %v", err)
	}

	// If audio is shorter than video, add silent frames
	if audioDuration < videoDuration {
		err = execFFMPEG("-y", "-i", audioPath, "-af", fmt.Sprintf("apad=whole_dur=%f", videoDuration), "-y", tempAudioPath)
		if err != nil {
			return fmt.Errorf("error padding audio with silence: %v", err)
		}
	} else if audioDuration > videoDuration {
		// If audio is longer, speed up the audio slightly
		atempoValue := audioDuration / videoDuration
		err = execFFMPEG("-y", "-i", audioPath, "-filter:a", fmt.Sprintf("atempo=%f", atempoValue), "-y", tempAudioPath)
		if err != nil {
			return fmt.Errorf("error adjusting audio speed: %v", err)
		}
	} else {
		// If audio and video have the same duration, use the original audio
		tempAudioPath = audioPath
	}

	// Merge adjusted audio with video
	err = execFFMPEG("-y", "-i", videoPath, "-i", tempAudioPath, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", "-map", "0:v", "-map", "1:a", outputPath)

	if err != nil {
		return fmt.Errorf("error merging video and audio: %v", err)
	}
	return nil
}

func MergeAllVideoSegmentsTogether(fileName string, segmentPaths []string) (string, error) {
	//Write all filepath into filelist.txt

	listFile := "filelist.txt"
	f, err := os.Create(listFile)
	if err != nil {
		log.Printf("Failed to create list file: %v", err)
		return "", fmt.Errorf("failed to create list file: %v", err)
	}
	defer f.Close()

	log.Println("Writing all video segment paths to list file...")

	for _, segmentPath := range segmentPaths {
		_, err = f.WriteString(fmt.Sprintf("file '%s'\n", segmentPath))
		if err != nil {
			log.Printf("Failed to write segment path to list file: %v", err)
			return "", fmt.Errorf("failed to write segment path to list file: %v", err)
		}
	}

	//Merge all segments into final output and store at /pkg/video_processing/final_output
	finalVideoDir := os.Getenv("PROCESSED_VIDEO_PATH")

	// 去掉 fileName 的 ".mp4" 後綴
	fileNameWithoutExt := strings.TrimSuffix(fileName, ".mp4")

	// 生成帶有 '_processed' 後綴的新名稱輸出檔名
	outputVideoName := fmt.Sprintf("%s_processed.mp4", fileNameWithoutExt)

	// 將新名稱用於最終輸出視頻的路徑
	outputVideo := path.Join(finalVideoDir, outputVideoName)

	log.Println("Running ffmpeg command to concat all segments from list file...")

	//Run FFmpeg "concat" to merge all segments together
	err = execFFMPEG("-y", "-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputVideo)
	if err != nil {
		log.Printf("Failed to merge video segments: %v", err)
		return "", fmt.Errorf("failed to merge video segments: %v", err)
	}

	log.Println("Successfully concat all segments from list file...")

	err = os.Remove(listFile)
	if err != nil {
		log.Printf("warning: failed to remove list file: %v", err)
	}

	log.Println("All tempfile has been removed")
	return outputVideo, nil
}
