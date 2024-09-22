package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
		id := r.FormValue(`path`)
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(`id missing`)
			return
		}

		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			w.WriteHeader(http.StatusExpectationFailed)
			log.Println(`invalid objectId`)
			return
		}

		var res Video
		err = collection.
			FindOne(context.Background(), bson.M{"_id": objectId}).
			Decode(&res)
		if err != nil {
			// ErrNoDocuments means that the filter did not match any documents in
			// the collection.
			if errors.Is(err, mongo.ErrNoDocuments) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}

		u, _ := url.Parse(fmt.Sprintf("%s:%s/video?id=%s", videoStorageHost, videoStoragePort, res.VideoPath))
		resp, err := http.Get(u.String())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add(contentType, "video/mp4")
		// use io.Copy for streaming.
		io.Copy(w, resp.Body)
	})

	http.ListenAndServe(fmt.Sprint(":", port), mux)
}
