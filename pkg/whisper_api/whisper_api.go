package whisper_api

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

type SentenceTimestamp struct {
	Sentence  string  `json:"Sentence"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

func CallWhisperAPI(apiKey string, audioReader io.Reader) (*WhisperResponse, []WordTimestamp, error) {
	url := "https://transcribe.whisperapi.com"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	file, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return nil, nil, err
	}

	_, err = io.Copy(file, audioReader)
	if err != nil {
		return nil, nil, err
	}

	_ = writer.WriteField("fileType", "mp3")
	_ = writer.WriteField("diarization", "false")
	_ = writer.WriteField("numSpeakers", "2")
	_ = writer.WriteField("language", "en")
	_ = writer.WriteField("task", "transcribe")

	err = writer.Close()
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Add("Authorization", "Bearer "+apiKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Whisper API responded with status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	var whisperResp WhisperResponse
	err = json.Unmarshal(body, &whisperResp)
	if err != nil {
		log.Printf("Error unmarshaling Whisper API response: %v", err)
		return nil, nil, err
	}

	log.Printf("Whisper API response unmarshaled successfully")
	/*
		// 迭代whisperResp.Segments並打印每個段落的Text字段
		for i, segment := range whisperResp.Segments {
			log.Printf("Segment %d Text: %s", i, segment.Text)
		}
	*/
	//log.Printf("Whisper API response text: %s", whisperResp.Text)
	// 生成基于单词的时间戳列表
	var wordTimestamps []WordTimestamp
	for _, segment := range whisperResp.Segments {
		for _, wholeWordTs := range segment.WholeWordTimestamps {
			wordTs := WordTimestamp{
				Word:        wholeWordTs.Word,
				StartTime:   wholeWordTs.StartTime,
				EndTime:     wholeWordTs.EndTime,
				Probability: wholeWordTs.Probability, // 假设这个字段也存在
			}
			wordTimestamps = append(wordTimestamps, wordTs)
		}
	}

	return &whisperResp, wordTimestamps, nil
}
