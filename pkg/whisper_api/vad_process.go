package whisper_api

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/1lann/dissonance/audio"
)

type ReaderToAudioStream struct {
	reader     io.Reader
	sampleRate int // 這是一個示例，您可能需要從其他來源獲取這個值
}

// 實現 audio.Stream 的 SampleRate 方法
func (ras *ReaderToAudioStream) SampleRate() int {
	return ras.sampleRate // 返回取樣率
}

// 實現 audio.Stream 的 Read 方法
func (ras *ReaderToAudioStream) Read(dst interface{}) (int, error) {
	// 將 interface{} 轉換為 []byte
	data, ok := dst.([]byte)
	if !ok {
		return 0, audio.ErrBufferTooLarge
	}

	// 使用內部 io.Reader 來填充數據
	return ras.reader.Read(data)
}

// ConvertToAudioStream 將 io.Reader 轉換為 audio.Stream
func ConvertToAudioStream(reader io.Reader) audio.Stream {
	return &ReaderToAudioStream{reader: reader}
}

// ProcessVADForSTT 用於處理 VAD 並將過濾後的音頻傳遞給 STT
func ProcessVADForSTT(audioReader io.Reader) (io.Reader, error) {
	// 初始化 VAD 過濾器
	vadFilter := NewFilter(0.5)

	// 轉換 audioReader 到 audio.Stream
	audioStream := ConvertToAudioStream(audioReader)

	// 過濾音頻數據
	filteredStream := vadFilter.Filter(audioStream)

	// 確保目錄存在
	outputDir := "../pkg/audio_processing/tmp/audio/filtered_by_vad"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Println("Failed to create output directory:", err)
		return nil, err
	}

	// 創建保存過濾後音頻的文件
	outputPath := filepath.Join(outputDir, "filtered_audio.mp3")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Println("Failed to create output file:", err)
		return nil, err
	}
	defer outputFile.Close()

	// 讀取過濾後的數據到一個 bytes.Buffer 和保存到文件
	var buf bytes.Buffer
	for {
		data := make([]byte, 4096)
		n, err := filteredStream.Read(data)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Failed to read from filteredStream:", err)
			return nil, err
		}
		buf.Write(data[:n])
		outputFile.Write(data[:n])
	}

	return bytes.NewReader(buf.Bytes()), nil
}
