package acapela_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
)

type LoginResponse struct {
	Token string `json:"token"`
}

type AcapelaResponse struct {
	Content []byte
}

func CallAcapelaAPI(text string, voice string) (AcapelaResponse, error) {
	// Define the login URL and the API endpoint
	loginURL := "https://www.acapela-cloud.com/api/login/"
	apiEndpoint := "https://www.acapela-cloud.com/api/command/"

	// 讀取環境變數
	email := os.Getenv("ACAPELA_EMAIL")
	password := os.Getenv("ACAPELA_PASSWORD")

	// 如果環境變數未設置，返回錯誤
	if email == "" || password == "" {
		return AcapelaResponse{}, fmt.Errorf("error: Missing email or password environment variable")
	}

	// Define the login credentials
	credentials := map[string]string{
		"email":    email,
		"password": password,
	}

	// Marshal the credentials into JSON
	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return AcapelaResponse{}, err
	}

	// Send a POST request to the login URL
	resp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(credentialsJSON))
	if err != nil {
		log.Printf("Error posting to Acapela login API: %v", err)
		return AcapelaResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Received status code %d from Acapela login API", resp.StatusCode)
		return AcapelaResponse{}, fmt.Errorf("error: Unable to login. Status code: %d", resp.StatusCode)
	}

	// 解析登入回應，檢查 Token 是否成功取得
	loginResponse := LoginResponse{}
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	if err != nil {
		log.Printf("Error decoding login response: %v", err)
		return AcapelaResponse{}, err
	}

	if loginResponse.Token == "" {
		log.Println("Received empty token from Acapela login API")
		return AcapelaResponse{}, fmt.Errorf("error: Received empty token")
	}

	// Define the data for the TTS request
	ttsData := map[string]string{
		"text":   text,
		"voice":  voice,
		"action": "create_file",
	}

	// Marshal the TTS data into JSON
	ttsDataJSON, err := json.Marshal(ttsData)
	if err != nil {
		return AcapelaResponse{}, err
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(ttsDataJSON))
	if err != nil {
		return AcapelaResponse{}, err
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+loginResponse.Token)

	// Send the request
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error posting to Acapela command API: %v", err)
		return AcapelaResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Received status code %d from Acapela command API", resp.StatusCode)
		return AcapelaResponse{}, fmt.Errorf("error: Unable to generate audio. Status code: %d", resp.StatusCode)
	}

	// Read the response content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return AcapelaResponse{}, err
	}

	return AcapelaResponse{Content: content}, nil
}

func ConvertTextToSpeechUsingAcapela(text string, voice string, segmentIndex int) (string, error) {
	// 使用提供的文字和語音調用Acapela API
	acapelaResp, err := CallAcapelaAPI(text, voice)
	if err != nil {
		log.Printf("Failed to convert text to speech using Acapela API: %v", err)
		return "", err
	}

	// 檢查返回的內容是否為mp3格式
	contentType := http.DetectContentType(acapelaResp.Content)
	if contentType != "audio/mpeg" {
		log.Println("The content is not in MP3 format")
		return "", fmt.Errorf("error: the content is not in MP3 format")
	}

	// 將返回的內容保存為mp3文件，使用segmentIndex生成唯一的檔名
	tempFileName := fmt.Sprintf("acapela_audio_segment_%d.mp3", segmentIndex)
	tempFilePath := path.Join("../pkg/video_processing/tmp/audio", tempFileName)
	err = saveAudioToFile(acapelaResp.Content, tempFileName) // 注意这里只需要传文件名
	if err != nil {
		log.Printf("Failed to save audio to file: %v", err)
		return "", err
	}

	return tempFilePath, nil
}
