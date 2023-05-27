package upload

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleUpload(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("video_file", "filename.mp4")
	if err != nil {
		t.Fatal(err)
	}
	_, err = fw.Write([]byte("test video file content"))
	if err != nil {
		t.Fatal(err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	// 模擬 WHISPER_API_KEY 環境變數
	os.Setenv("WHISPER_API_KEY", "test_api_key")

	worker := Worker{
		JobQueue: make(chan Job, 1), // 創建一個帶有緩衝區的 channel
	}
	// 在後台讀取 job，以便測試可以完成
	go func() {
		<-worker.JobQueue
	}()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HandleUpload(w, r, worker)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

}
