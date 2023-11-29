#!/bin/bash

# 安裝 Go 1.21.4
GO_VERSION="1.21.4"
wget https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz
rm go$GO_VERSION.linux-amd64.tar.gz

# 添加 Go 環境變數
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc

# 安裝 FFmpeg
sudo apt update
sudo apt install -y ffmpeg

# 添加自訂環境變數到 ~/.bashrc
echo "export WHISPER_API_KEY=6BHKUBPE5S43JPPKTBXJGFYW3XNA5S7G" >> ~/.bashrc
echo "export ACAPELA_EMAIL=cathyhaien@hotmail.com" >> ~/.bashrc
echo "export ACAPELA_PASSWORD=jHFXuBxqZNBJTl" >> ~/.bashrc
echo "export VIDEO_PROCESSING_PORT=30017" >> ~/.bashrc
echo "export VIDEO_PROCESSING_LOG_PATH=workingProgress.log" >> ~/.bashrc
echo "export UNPROCESSED_VIDEO_PATH=/home/shared/unprocessed_videos" >> ~/.bashrc
echo "export PROCESSED_VIDEO_PATH=/home/shared/processed_videos" >> ~/.bashrc

# 創建資料夾並設定權限
mkdir -p /home/shared/unprocessed_videos
mkdir -p /home/shared/processed_videos
chmod 777 /home/shared/unprocessed_videos
chmod 777 /home/shared/processed_videos

# 重載 ~/.bashrc
source ~/.bashrc

echo "安裝和設定完成。"
