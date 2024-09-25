package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	contentType = "Content-Type"
)

type Video struct {
	ID        primitive.ObjectID `bson:"_id"`
	VideoPath string             `bson:"videoPath"`
}

func failOnMissingEnvironmentVariable(variableName, failureMessage string) string {
	value, found := os.LookupEnv(variableName)
	if !found {
		log.Fatal(failureMessage)
	}

	return value
}

func getPathFromObjectID(w http.ResponseWriter, id string, collection *mongo.Collection) *string {
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		slog.Error(`id missing`)
		return nil
	}

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.WriteHeader(http.StatusExpectationFailed)
		slog.Error(`invalid objectId`)
		return nil
	}

	var video Video
	err = collection.
		FindOne(context.Background(), bson.M{"_id": objectId}).
		Decode(&video)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if errors.Is(err, mongo.ErrNoDocuments) {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}

		slog.Info(`unknown error`, `MongoErr`, err)
		return nil
	}

	slog.Info(`S3 response`, `objectID`, objectId.String(), `ID`, video.ID, `path`, video.VideoPath)

	return &video.VideoPath
}

func main() {
	port := failOnMissingEnvironmentVariable(`PORT`, `Please specify the port number for the HTTP server with the environment variable PORT.`)
	videoStorageHost := failOnMissingEnvironmentVariable(`VIDEO_STORAGE_HOST`, `Please specify the video storage host name with the environment variable VIDEO_STORAGE_HOST.`)
	videoStoragePort := failOnMissingEnvironmentVariable(`VIDEO_STORAGE_PORT`, `Please specify the video storage port number with the environment variable VIDEO_STORAGE_PORT.`)
	dbhost := failOnMissingEnvironmentVariable(`DBHOST`, `Please specify the database host for the id->path mapping with the environment variable DBHOST.`)
	dbname := failOnMissingEnvironmentVariable(`DBNAME`, `Please specify the database name for the id->path mapping with the environment variable DBHOST.`)

	// Connect to Mongo
	// https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo
	clientOpts := options.Client().
		ApplyURI(dbhost)
	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		log.Fatal("failed to connect to MongoDB", err)
	}
	collection := client.Database(dbname).Collection(`videos`)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /video", func(w http.ResponseWriter, r *http.Request) {
		videoPath := getPathFromObjectID(w, r.FormValue(`id`), collection)
		if videoPath == nil {
			slog.Info(`video-storage`, `id`, `not found`)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		videoStorageURL := fmt.Sprintf("%s:%s/video?id=%s", videoStorageHost, videoStoragePort, *videoPath)
		req, _ := http.NewRequest(http.MethodGet, videoStorageURL, nil)
		req.Header = r.Header

		// Retrieve videoStorageURL.
		resp, err := http.DefaultClient.Do(req)
		// This error check alone won't check for 404s, so we need to
		// ensure we fail on any non-OK response.
		if err != nil || resp.StatusCode != http.StatusOK {
			slog.Info(`video-storage`, `retrieval error`, err, `url`, videoStorageURL, `addr`, resp.Request.RemoteAddr)
			w.WriteHeader(http.StatusFailedDependency)
			return
		}
		defer resp.Body.Close()

		w.Header().Add(contentType, "video/mp4")
		io.Copy(w, resp.Body)
	})

	http.ListenAndServe(fmt.Sprint(":", port), mux)
}
