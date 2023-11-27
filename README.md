# acapelatts_go

#-t 參數用來自定義image之tag
docker build -t video-processor:latest .

container啟動指令:docker run -d -v /home/shared/video_processing_log:/app/log -v /home/shared/unprocessed_videos:/home/shared/unprocessed_videos -v /home/shared/processed_videos:/home/shared/processed_videos -p 30016:30016 --env-file .env --name video-processor-go video-processor:latest
