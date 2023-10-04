# 使用官方的 Golang image 作為基礎 image
FROM golang:1.21.1 as builder

# 設定工作目錄
WORKDIR /app

# 複製 go mod 和 sum 檔案
COPY go.mod go.sum ./

# 下載所有dependency
RUN go mod download

# 複製source code
COPY . .

# 編譯應用程式
RUN go build -o /video-processing ./cmd/main.go

# 使用 Ubuntu 作為基礎 image
FROM ubuntu:latest

# 更新套件並重新安裝 CA 證書並安裝ffmpeg
RUN apt-get update && apt-get install --reinstall -y ca-certificates && apt-get install -y ffmpeg

# 複製編譯後的app到當前目錄
COPY --from=builder /video-processing /video-processing

# 運行app
CMD ["/video-processing"]


#container啟動指令:docker run -v /home/shared/videos:../pkg/video_processing/final_videos -p 30016:30016 --name video-processor-go --env-file .env video-processing:1.0 