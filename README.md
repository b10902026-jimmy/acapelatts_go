# Video Upload and Processing Service

這是一個使用 Go 語言開發的視頻處理服務，主要功能包括從視頻中提取語音、將語音轉換為文本、生成字幕文件，並將視頻分割成基於字幕的多個片段。

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

## 安裝

首先，克隆此存儲庫到您的本地機器：

```bash
git clone https://github.com/your-username/your-repository.git
cd your-repository



#-t 參數用來自定義image之tag
docker build -t video-processor:latest .

container啟動指令:docker run -d -v /home/shared/video_processing_log:/app/log -v /home/shared/unprocessed_videos:/home/shared/unprocessed_videos -v /home/shared/processed_videos:/home/shared/processed_videos -p 30016:30016 --env-file .env --name video-processor-go video-processor:latest
