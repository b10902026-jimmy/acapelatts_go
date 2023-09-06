package whisper_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

// 定義Whisper API的響應結構
type WhisperResponse struct {
	Text     string `json:"text"`
	Language string `json:"language"`
	Segments []struct {
		Start               float64         `json:"start"`
		End                 float64         `json:"end"`
		Text                string          `json:"text"`
		WholeWordTimestamps []WordTimestamp `json:"whole_word_timestamps"`
	} `json:"segments"`
}

// 定義單個單詞的時間戳結構
type WordTimestamp struct {
	Word        string  `json:"word"`
	StartTime   float64 `json:"start"`
	EndTime     float64 `json:"end"`
	Probability float64 `json:"probability"`
}

// 定義句子的時間戳結構
type SentenceTimestamp struct {
	Sentence  string  `json:"Sentence"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

type WhisperAndWordTimestamps struct {
	WhisperResp    *WhisperResponse
	WordTimestamps []WordTimestamp
}

func CallWhisperAPI(apiKey string, audioReader io.Reader) (*WhisperAndWordTimestamps, error) {

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
	log.Printf("Status Code: %d", res.StatusCode)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Whisper API responded with status code: %d", res.StatusCode)
		return nil, errors.New("received non-200 status code")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Printf("Response Body: %s", string(body))
	var whisperResp WhisperResponse
	err = json.Unmarshal(body, &whisperResp)
	if err != nil {
		log.Printf("Error unmarshaling Whisper API response: %v", err)
		return nil, err
	}

	log.Printf("Whisper API response unmarshaled successfully")
	/*
		// 迭代whisperResp.Segments並打印每個段落的Text字段
		for i, segment := range whisperResp.Segments {
			log.Printf("Segment %d Text: %s", i, segment.Text)
		}
	*/

	log.Printf("Whisper API response text: %+v", whisperResp)

	//Define the content of the sentenceTimestamps for video
	/*sentenceTimestamps := []SentenceTimestamp{}
	for _, segment := range whisperResp.Segments {
		sentenceTimestamp := SentenceTimestamp{
			Sentence:  segment.Text,
			StartTime: segment.Start,
			EndTime:   segment.End,
		}
		sentenceTimestamps = append(sentenceTimestamps, sentenceTimestamp)
	}*/

	// Populate wordTimestamps
	wordTimestamps := []WordTimestamp{}
	for _, segment := range whisperResp.Segments {
		for _, wordTimestamp := range segment.WholeWordTimestamps {
			simplifiedWordTimestamp := WordTimestamp{
				Word:      wordTimestamp.Word,
				StartTime: wordTimestamp.StartTime,
				EndTime:   wordTimestamp.EndTime,
			}
			wordTimestamps = append(wordTimestamps, simplifiedWordTimestamp)
		}
	}

	return &WhisperAndWordTimestamps{
		WhisperResp:    &whisperResp,
		WordTimestamps: wordTimestamps,
	}, nil
}
