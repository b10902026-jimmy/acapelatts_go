package audio_processing

import (
	"fmt"
	"log"
	"os"
	"path"
)

func SaveAudioToFile(content []byte, filename string) error {
	// 拼接檔案路径
	filePath := path.Join("../pkg/audio_processing/test_files", filename)

	// 建立檔案
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("error creating file: %v", err)
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// 寫入音檔內容
	_, err = file.Write(content)
	if err != nil {
		log.Printf("error writing to file: %v", err)
		return fmt.Errorf("error writing to file: %v", err)
	}

	log.Printf("Succesfully save the mp3 file from Acapela to the local 'test_files'. ")

	return nil
}
