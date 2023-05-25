package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

// 定義Whisper API的響應結構
type WhisperResponse struct {
	Text     string `json:"text"`
	Segments []struct {
		WholeWordTimestamps []struct {
			Word  string  `json:"word"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"whole_word_timestamps"`
	} `json:"segments"`
}

type WordTimestamp struct {
	Word      string  `json:"word"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

func CallWhisperAPI(apiKey string, audioReader io.Reader) (*WhisperResponse, error) {
	url := "https://transcribe.whisperapi.com"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	file, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(file, audioReader)
	if err != nil {
		return nil, err
	}

	_ = writer.WriteField("fileType", "mp3")
	_ = writer.WriteField("diarization", "false")
	_ = writer.WriteField("numSpeakers", "2")
	_ = writer.WriteField("language", "en")
	_ = writer.WriteField("task", "transcribe")

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Add("Authorization", "Bearer "+apiKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Whisper API responded with status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var whisperResp WhisperResponse
	err = json.Unmarshal(body, &whisperResp)
	if err != nil {
		log.Printf("Error unmarshaling Whisper API response: %v", err)
		return nil, err
	}
	log.Printf("Whisper API response unmarshaled successfully")
	log.Printf("Whisper API response text: %s", whisperResp.Text)
	return &whisperResp, nil
}
