package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
)

func failOnMissingEnvironmentVariable(variableName, failureMessage string) string {
	value, found := os.LookupEnv(variableName)
	if !found {
		log.Fatal(failureMessage)
	}

	return value
}

func main() {
	port := failOnMissingEnvironmentVariable(`PORT`, `Please specify the port number for the HTTP server with the environment variable PORT.`)
	videoStorageHost := failOnMissingEnvironmentVariable(`VIDEO_STORAGE_HOST`, `Please specify the video storage host name with the environment variable VIDEO_STORAGE_HOST.`)
	videoStoragePort := failOnMissingEnvironmentVariable(`VIDEO_STORAGE_PORT`, `Please specify the video storage port number with the environment variable VIDEO_STORAGE_PORT.`)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /video", func(w http.ResponseWriter, r *http.Request) {
		videoPath := "/video?path=SampleVideo_1280x720_1mb.mp4"
		videoStorageURL := fmt.Sprintf(`%s:%s%s`, videoStorageHost, videoStoragePort, videoPath)
		req, _ := http.NewRequest(http.MethodGet, videoStorageURL, nil)
		req.Header = r.Header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusFailedDependency)
			log.Printf("couldn't retrieve %s from %s", videoPath, videoStorageHost)
			return
		}
		defer resp.Body.Close()

		// use io.Copy for streaming.
		io.Copy(w, resp.Body)
	})

	slog.Info(`video-storage online`)
	http.ListenAndServe(fmt.Sprint(":", port), mux)
}
