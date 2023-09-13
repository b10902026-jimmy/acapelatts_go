package audio_processing

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
)

func StreamedExtractAudioFromVideo(filePath string) (io.Reader, error) {
	// 命令設置
	cmd := exec.Command("ffmpeg", "-i", filePath, "-f", "mp3", "-vn", "pipe:1")
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return nil, err
	}

	// 保存MP3文件到當前工作目錄
	mp3File, err := os.Create("output.mp3")
	if err != nil {
		log.Printf("Failed to create MP3 file: %v", err)
		return nil, err
	}
	defer mp3File.Close()

	var buf bytes.Buffer
	go func() {
		// 同時將音頻數據寫入Buffer和MP3文件
		multiWriter := io.MultiWriter(&buf, mp3File)
		io.Copy(multiWriter, stdoutPipe)
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
