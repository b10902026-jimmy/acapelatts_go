package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/xfrr/goffmpeg/transcoder"
)

func ExtractAudioFromVideo(inputFile io.Reader) (io.Reader, error) {
	inputFileTemp, err := ioutil.TempFile("", "input-*.mp4")
	if err != nil {
		return nil, fmt.Errorf("Error creating temporary input file: %v", err)
	}
	inputFilePath := inputFileTemp.Name()
	defer os.Remove(inputFilePath)

	outputFileTemp, err := ioutil.TempFile("", "output-*.mp3")
	if err != nil {
		return nil, fmt.Errorf("Error creating temporary output file: %v", err)
	}
	outputFilePath := outputFileTemp.Name()
	defer os.Remove(outputFilePath)

	// 將上傳的文件保存到臨時文件中
	inputFileBytes, err := ioutil.ReadAll(inputFile)
	if err != nil {
		return nil, fmt.Errorf("Error reading input file: %v", err)
	}

	err = ioutil.WriteFile(inputFilePath, inputFileBytes, 0644)
	if err != nil {
		return nil, fmt.Errorf("Error writing input file to disk: %v", err)
	}

	// 初始化轉碼器
	trans := new(transcoder.Transcoder)

	// 配置轉碼器
	err = trans.Initialize(inputFilePath, outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("Error initializing transcoder: %v", err)
	}

	// 開始轉碼
	done := trans.Run(false)

	// 等待轉碼完成
	err = <-done
	if err != nil {
		return nil, fmt.Errorf("Error transcoding: %v", err)
	}

	// 讀取輸出音訊文件
	outputFileBytes, err := ioutil.ReadFile(outputFilePath)
	if err != nil {
		log.Printf("Error reading output file: %v", err)
		return nil, fmt.Errorf("Error reading output file: %v", err)
	}
	log.Printf("Audio extracted successfully")

	return bytes.NewReader(outputFileBytes), nil
}
