package acapela_api

import (
	"fmt"
	"log"
	"os"
	"path"
)

func saveAudioToFile(content []byte, tempAudioFilePath string) error {

	// 確認路徑存在，如果不存在則建立資料夾
	dirPath := path.Dir(tempAudioFilePath)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			log.Printf("error creating directory: %v", err)
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	// 建立檔案
	file, err := os.Create(tempAudioFilePath)
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

	return nil
}
