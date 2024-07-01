package main

import (
	"context"
	"fmt"
	"io"
	"log"
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
	awsEndpoint string = "http://127.0.0.1:9000"
	bucketName  string

	s3svc *s3.Client
)

func init() {
	bucketName = os.Getenv("S3_BUCKET")

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
		Credentials:      credentials.NewStaticCredentialsProvider("root", "changeme", ""),
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
		input := &s3.GetObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("SampleVideo_1280x720_1mb.mp4"),
		}
		result, err := s3svc.GetObject(context.Background(), input)
		if err != nil {
			w.WriteHeader(404)
			log.Printf("%s not found.", *input.Key)
			return
		}
		defer result.Body.Close()

		io.Copy(w, result.Body)
	})

	http.ListenAndServe(fmt.Sprint(":", port), mux)
}
