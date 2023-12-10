# 影片上傳並將人聲轉換為機器聲之音訊轉換服務

這是一個使用 Go 語言開發的視頻處理服務，主要功能包括從視頻中提取語音、將語音轉換為文本、生成字幕文件，並將視頻分割成基於字幕的多個片段。各個片段分別平行處理，將人聲替換成AI語音後，再將所有處理過的片段結合輸出。

## 功能

- 從視頻中提取語音
- 使用 STT API 將語音轉換為文本
- 創建 SRT 字幕文件
- 根據字幕將視頻分割成片段
- 使用 TTS API 將文本轉換為語音
- 將合成語音替換原視頻中的語音軌
- 將處理後的視頻片段重新組合成完整視頻

## 前置要求

在運行此服務之前，您需要確保已安裝以下軟件：

- [Go](https://golang.org/dl/) (版本 1.13 或更高)
- [FFmpeg](https://ffmpeg.org/download.html)

如果使用linux系統，運行以下命令安裝FFmpeg:

```bash
sudo apt update
sudo apt install ffmpeg -y
```

- 請參考.env.example進行環境變數的配置，並將.env檔建立於程式的根目錄


## 使用

首先，克隆此存儲庫到您的本地機器：

```bash
git clone https://github.com/b10902026-jimmy/videoUploadAndProcessing_go
cd videoUploadAndProcessing_go/cmd
go run main.go
```
你可以在'cmd'目錄底下的'workingProgress.log'查看程式的運行日誌

## 結構說明

 項目的目錄結構如下：

``` bash
.
├── cmd
│   └── main.go
├── pkg
│   ├── acapela_api
│   ├── upload
│   ├── video_processing
│   └── whisper_api
└── README.md

```

- 'cmd/main.go' 是應用程序的入口點。
- 'pkg' 目錄包含所有與視頻處理相關的業務邏輯。


## 若要用此專案在docker容器上運行: 

### 根據目錄中的dockerfile構建映像並且啟動容器:
```bash
docker-compose up -d --build
```

### 若不需構建映像檔（主機上已有video-processor映像檔）：
```bash
docker-compose up -d 
```

-----------------------------------------

# Video Upload and Processing Service

This is a video processing service developed using the Go language. Its main features include extracting audio from videos, converting the audio to text, generating subtitle files, and dividing the video into multiple segments based on subtitles. Each segment is processed in parallel, replacing human voices with AI-generated voices, and then combining all the processed segments for output.

## Features

- Extract voice from video
- Use STT API to convert voice to text
- Create SRT subtitle files
- Split the video into segments based on subtitles
- Use TTS API to convert text to voice
- Replace the original voice track in the video with synthesized voice
- Reassemble the processed video segments into a complete video

## Prerequisites

Before running this service, ensure you have installed the following software:

- [Go](https://golang.org/dl/) (version 1.13 or higher)
- [FFmpeg](https://ffmpeg.org/download.html)

If you're using a Linux system, run the following command to install FFmpeg:

```bash
sudo apt update
sudo apt install ffmpeg -y
```

## Usage

First, clone this repository to your local machine:

``` bash
git clone https://github.com/b10902026-jimmy/videoUploadAndProcessing_go
cd videoUploadAndProcessing_go/cmd
go run main.go

```
You can view the program's running log in the workingProgress.log under the cmd directory.

## Structure Description

The project's directory structure is as follows:

``` bash
.
├── cmd
│   └── main.go
├── pkg
│   ├── acapela_api
│   ├── upload
│   ├── video_processing
│   └── whisper_api
└── README.md

```

- 'cmd/main.go' is the entry point of the application.
- The 'pkg' directory contains all business logic related to video processing.


## Running this Project in a Docker Container

### Build the Docker image from Dockerfile in the directory and run the container: 

```bash
docker-compose up -d --build
```

### If doesn't need to build the Docker image （Docker image video-processor is already exist）：
```bash
docker-compose up -d 
```





