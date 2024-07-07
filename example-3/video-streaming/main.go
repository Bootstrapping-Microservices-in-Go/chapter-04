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
	contentLength = "Content-Length"
	contentType   = "Content-Type"
)

type Video struct {
	ID        primitive.ObjectID `bson:"_id"`
	VideoPath string             `bson:"videoPath"`
}

func main() {
	port := os.Getenv(`PORT`)
	dbhost := os.Getenv(`DBHOST`)
	dbname := os.Getenv(`DBNAME`)
	videoStorageHost := os.Getenv(`VIDEO_STORAGE_HOST`)
	videoStoragePort := os.Getenv(`VIDEO_STORAGE_PORT`)

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
		id := r.FormValue(`id`)
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

		u, _ := url.Parse(fmt.Sprintf("http://%s:%s/video?id=%s", videoStorageHost, videoStoragePort, res.VideoPath))
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
