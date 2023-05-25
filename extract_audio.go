package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/xfrr/goffmpeg/transcoder"
)

func ExtractAudioFromVideo(inputFile io.Reader) (io.Reader, error) {
	inputFileTemp, err := os.CreateTemp("", "input-*.mp4")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary input file: %v", err)
	}
	inputFilePath := inputFileTemp.Name()
	defer os.Remove(inputFilePath)

	outputFileTemp, err := os.CreateTemp("", "output-*.mp3")
	if err != nil {
		os.Remove(inputFilePath) // 刪除輸入暫存檔案
		return nil, fmt.Errorf("error creating temporary output file: %v", err)
	}
	outputFilePath := outputFileTemp.Name()
	defer os.Remove(outputFilePath)

	// 將上傳的文件保存到臨時文件中
	_, err = io.Copy(inputFileTemp, inputFile) // 直接將 inputFile 的資料流寫入到你的臨時檔案
	if err != nil {
		os.Remove(inputFilePath)  // 刪除輸入暫存檔案
		os.Remove(outputFilePath) // 刪除輸出暫存檔案
		return nil, fmt.Errorf("error writing input file to disk: %v", err)
	}

	// 初始化轉碼器
	trans := new(transcoder.Transcoder)

	// 配置轉碼器
	err = trans.Initialize(inputFilePath, outputFilePath)
	if err != nil {
		os.Remove(inputFilePath)  // 刪除輸入暫存檔案
		os.Remove(outputFilePath) // 刪除輸出暫存檔案
		return nil, fmt.Errorf("error initializing transcoder: %v", err)
	}

	// 開始轉碼
	done := trans.Run(false)

	// 等待轉碼完成
	err = <-done
	if err != nil {
		os.Remove(inputFilePath)  // 刪除輸入暫存檔案
		os.Remove(outputFilePath) // 刪除輸出暫存檔案
		return nil, fmt.Errorf("error transcoding: %v", err)
	}

	// 讀取輸出音訊文件
	outputFileBytes, err := os.ReadFile(outputFilePath)
	if err != nil {
		os.Remove(inputFilePath)  // 刪除輸入暫存檔案
		os.Remove(outputFilePath) // 刪除輸出暫存檔案
		return nil, fmt.Errorf("error reading output file: %v", err)
	}
	log.Printf("Audio extracted successfully")

	os.Remove(inputFilePath)
	os.Remove(outputFilePath)

	return bytes.NewReader(outputFileBytes), nil
}
