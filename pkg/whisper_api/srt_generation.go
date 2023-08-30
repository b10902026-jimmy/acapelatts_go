package whisper_api

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SRTSegment struct {
	StartTime string
	EndTime   string
	Text      string
}

func CreateSRTFile(whisperResp *WhisperResponse) error {
	// 指定SRT文件的輸出路徑
	outputPath := "../pkg/audio_processing/tmp/subtitles/output.srt"

	// 檢查並創建目錄（如果不存在）
	outputDir := filepath.Dir(outputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	// 打開SRT文件進行寫入
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating SRT file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// 迭代每個片段並寫入SRT格式
	for i, segment := range whisperResp.Segments {
		// SRT索引
		fmt.Fprintf(writer, "%d\n", i+1)

		// 轉換開始和結束時間
		startTime := secondsToSRTFormat(segment.Start)
		endTime := secondsToSRTFormat(segment.End)
		fmt.Fprintf(writer, "%s --> %s\n", startTime, endTime)

		// 寫入句子
		fmt.Fprintln(writer, segment.Text)

		/*
			// 寫入單詞時間戳
			for _, wordTimestamp := range segment.WholeWordTimestamps {
				wordStartTime := secondsToSRTFormat(wordTimestamp.StartTime)
				wordEndTime := secondsToSRTFormat(wordTimestamp.EndTime)
				fmt.Fprintf(writer, "%s --> %s: %s\n", wordStartTime, wordEndTime, wordTimestamp.Word)
			}
		*/

		// 添加空行分隔
		fmt.Fprintln(writer, "")
	}

	return writer.Flush()
}

// secondsToSRTFormat將秒轉換為SRT格式的時間戳
func secondsToSRTFormat(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	seconds = float64(int(seconds)%60) + (seconds - float64(int(seconds)))
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, seconds)
}

func ReadSRTFile(filePath string) ([]SRTSegment, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var segments []SRTSegment
	scanner := bufio.NewScanner(file)
	var currentSegment SRTSegment
	var readingText bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if readingText {
				segments = append(segments, currentSegment)
				currentSegment = SRTSegment{}
				readingText = false
			}
			continue
		}

		if !readingText {
			if strings.Contains(line, "-->") {
				times := strings.Split(line, "-->")
				if len(times) != 2 {
					return nil, errors.New("invalid time format")
				}
				currentSegment.StartTime = strings.TrimSpace(times[0])
				currentSegment.EndTime = strings.TrimSpace(times[1])
			} else if _, err := fmt.Sscanf(line, "%d", new(int)); err == nil {
				// This is the index line, do nothing
			} else {
				// This is the text line
				currentSegment.Text = line
				readingText = true
			}
		} else {
			currentSegment.Text += " " + line
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return segments, nil
}
