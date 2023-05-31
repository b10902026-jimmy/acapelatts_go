package audio_processing

import (
	"io"
	"log"
	"net/http"
	"os"
	"testing"
)

func TestExtractAudioFromVideo(t *testing.T) {
	// 打開測試影片
	inputFile, err := os.Open("./test_files/test_video.mp4")
	if err != nil {
		log.Println(err)
		t.Fatalf("failed to open video file: %v", err)
	}
	defer inputFile.Close()

	// 呼叫解析函式提取音訊
	output, err := ExtractAudioFromVideo(inputFile)
	if err != nil {
		t.Fatalf("ExtractAudioFromVideo failed: %v", err)
	}

	// 讀取輸出檔案
	var outputBytes []byte
	outputBytes, err = io.ReadAll(output)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// 檢查輸出是否為空
	if len(outputBytes) == 0 {
		t.Errorf("expected output to be non-empty, but it was empty")
	}

	// 檢查輸出是否為mp3
	contentType := http.DetectContentType(outputBytes)
	if contentType != "audio/mpeg" {
		t.Errorf("output file is not in MP3 format, got: %s", contentType)
	}
}
