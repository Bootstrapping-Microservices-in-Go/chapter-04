package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
)

const (
	contentLength = "Content-Length"
	contentType   = "Content-Type"
)

func main() {
	port, found := os.LookupEnv(`PORT`)
	if !found {
		log.Fatal(`Please specify the port number for the HTTP server with the environment variable PORT.`)
	}
	videoStorageHost := os.Getenv(`VIDEO_STORAGE_HOST`)
	videoStoragePort := os.Getenv(`VIDEO_STORAGE_PORT`)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /video", func(w http.ResponseWriter, r *http.Request) {
		videoPath := "/video?path=SampleVideo_1280x720_1mb.mp4"
		videoStorageURL := fmt.Sprintf(`%s:%s%s`, videoStorageHost, videoStoragePort, videoPath)
		log.Printf(`Attempting to retrieve %s`, videoStorageURL)
		req, _ := http.NewRequest(http.MethodGet, videoStorageURL, nil)
		req.Header = r.Header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusFailedDependency)
			log.Printf("couldn't retrieve %s", videoPath)
			return
		}
		defer resp.Body.Close()

		// use io.Copy for streaming.
		io.Copy(w, resp.Body)
	})

	slog.Info(`video-storage online`)
	http.ListenAndServe(fmt.Sprint(":", port), mux)
}
