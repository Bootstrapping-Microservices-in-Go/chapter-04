package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CustomEvent for lambda
type CustomEvent struct {
	ID   string
	Name string
}

var (
	awsRegion   string = "us-east-1"
	awsEndpoint string = "http://minio:9000"
	bucketName  string

	s3svc *s3.Client
)

func init() {
	bucketName = os.Getenv("BUCKET")
	minioUser := "steven"
	minioPassword := "changeme"

	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           awsEndpoint,
			SigningRegion: awsRegion,
		}, nil
	})

	awsCfg := aws.Config{
		Region:           awsRegion,
		EndpointResolver: customResolver,
		Credentials: credentials.NewStaticCredentialsProvider(
			minioUser,
			minioPassword,
			""),
	}
	s3svc = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
}

func main() {
	port, found := os.LookupEnv(`PORT`)
	if !found {
		log.Fatal(`Please specify the port number for the HTTP server with the environment variable PORT.`)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /video", func(w http.ResponseWriter, r *http.Request) {
		path := r.FormValue(`path`)
		log.Println(`Attempting to retrieve `, path)
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(path),
		}
		result, err := s3svc.GetObject(context.Background(), input)
		slog.Info(
			"s3svc.GetObject",
			`bucket`, *input.Bucket,
			`key`, *input.Key,
			`error`, err,
		)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			log.Printf("%s not found.", path)
			_, lbErr := s3svc.ListBuckets(context.TODO(), nil)
			slog.Info(`err != nil`, `error`, lbErr)
			log.Print(err.Error())
			return
		}
		defer result.Body.Close()

		io.Copy(w, result.Body)
	})

	slog.Info(`video-storage online`, `port`, port, `bucketName`, bucketName, `host`, awsEndpoint)
	http.ListenAndServe(fmt.Sprint(":", port), mux)
}
