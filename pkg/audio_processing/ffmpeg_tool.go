package audio_processing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type VideoMetadata struct {
	BitRate         string `json:"bit_rate"`
	FrameRate       string `json:"r_frame_rate"`
	AudioSampleRate string `json:"sample_rate"`
	AudioChannels   int    `json:"channels"`
	// 添加其他所需的欄位
}

func ExecFFMPEG(args ...string) error {
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %v, output: %s", err, output)
	}
	return nil
}

func GetVideoMetadata(filePath string) (VideoMetadata, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return VideoMetadata{}, err
	}

	var result map[string]interface{}
	json.Unmarshal(output, &result)

	streams := result["streams"].([]interface{})
	videoStream := streams[0].(map[string]interface{})
	audioStream := streams[1].(map[string]interface{})

	metadata := VideoMetadata{
		BitRate:         videoStream["bit_rate"].(string),
		FrameRate:       videoStream["r_frame_rate"].(string),
		AudioSampleRate: audioStream["sample_rate"].(string),
		AudioChannels:   int(audioStream["channels"].(float64)),
	}
	return metadata, nil
}

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
