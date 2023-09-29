package video_processing

import (
	"fmt"
	"os/exec"
	"strings"
)

func GetCodecs(filePath string) (string, string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	videoCodec, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting video codec: %v", err)
	}

	cmd = exec.Command("ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	audioCodec, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting audio codec: %v", err)
	}

	return strings.TrimSpace(string(videoCodec)), strings.TrimSpace(string(audioCodec)), nil
}
