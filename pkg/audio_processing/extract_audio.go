package audio_processing

import (
	"bytes"
	"io"
	"log"
	"os/exec"
)

func StreamedExtractAudioFromVideo(filePath string) (io.Reader, error) {
	cmd := exec.Command("ffmpeg", "-i", filePath, "-f", "mp3", "-vn", "pipe:1")
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return nil, err
	}

	var buf bytes.Buffer
	go func() {
		io.Copy(&buf, stdoutPipe)
	}()

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start ffmpeg: %v", err)
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("Failed to wait for ffmpeg: %v", err)
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}
