package whisper_api

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type SRTSegment struct {
	StartTime float64
	EndTime   float64
	Text      string
}

func StreamedCreateSRTFile(whisperAndWordTimestamps *WhisperAndWordTimestamps) (io.Reader, error) {
	// 檢查並創建目錄（如果不存在）
	outputPath := "../pkg/audio_processing/tmp/subtitles/output.srt"
	outputDir := filepath.Dir(outputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	// 創建一個緩衝區用於寫入SRT資料
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// 迭代每個片段並寫入SRT格式
	for i, segment := range whisperAndWordTimestamps.WhisperResp.Segments {
		// SRT索引
		fmt.Fprintf(writer, "%d\n", i+1)

		// 轉換開始和結束時間
		startTime := secondsToSRTFormat(segment.Start)
		endTime := secondsToSRTFormat(segment.End)
		fmt.Fprintf(writer, "%s --> %s\n", startTime, endTime)

		// 寫入句子
		fmt.Fprintln(writer, segment.Text)

		// 添加空行分隔
		fmt.Fprintln(writer, "")
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush the buffer: %v", err)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

// secondsToSRTFormat將秒轉換為SRT格式的時間戳
func secondsToSRTFormat(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	seconds = float64(int(seconds)%60) + (seconds - float64(int(seconds)))
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, seconds)
}

func CreateWholeWordTimestampsFile(whisperAndWordTimestamps *WhisperAndWordTimestamps) (string, error) {
	outputPath := "../pkg/audio_processing/tmp/subtitles/wholeWordTimestamps.srt"

	outputDir := filepath.Dir(outputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("error creating wholeWordTimestamps file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, wordTimestamp := range whisperAndWordTimestamps.WordTimestamps {
		//fmt.Fprintf(writer, "%d\n", i+1) // SRT序號
		fmt.Fprintf(writer, "%f --> %f", wordTimestamp.StartTime, wordTimestamp.EndTime)
		fmt.Fprintf(writer, "%s", wordTimestamp.Word) // 實際的字詞和一個空行
	}

	err = writer.Flush()
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func ReadSRTStream(srtReader io.Reader) ([]SRTSegment, error) {
	var segments []SRTSegment
	scanner := bufio.NewScanner(srtReader)
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
				startTime, err := srtTimeToSeconds(strings.TrimSpace(times[0]))
				if err != nil {
					return nil, err
				}
				endTime, err := srtTimeToSeconds(strings.TrimSpace(times[1]))
				if err != nil {
					return nil, err
				}
				currentSegment.StartTime = startTime
				currentSegment.EndTime = endTime
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

func srtTimeToSeconds(srtTime string) (float64, error) {
	parts := strings.Split(srtTime, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid SRT time format: %s", srtTime)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour format in SRT time: %s", parts[0])
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute format in SRT time: %s", parts[1])
	}

	secondsParts := strings.Split(parts[2], ".")
	if len(secondsParts) != 2 {
		return 0, fmt.Errorf("invalid seconds and milliseconds format in SRT time: %s", parts[2])
	}

	seconds, err := strconv.Atoi(secondsParts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid second format in SRT time: %s", secondsParts[0])
	}

	milliseconds, err := strconv.Atoi(secondsParts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid millisecond format in SRT time: %s", secondsParts[1])
	}
	return float64(hours*3600+minutes*60+seconds) + float64(milliseconds)/1000.0, nil
}
