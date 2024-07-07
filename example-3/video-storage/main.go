package main

import (
	"context"
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
	awsRegion     string = "us-east-1"
	awsEndpoint   string = "http://minio:9000"
	bucketName    string
	minioUser     string
	minioPassword string

	s3svc *s3.Client
)

func init() {
	bucketName = os.Getenv("BUCKET")
	minioUser = "steven"
	minioPassword = "changeme"

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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /video", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue(`id`)
		log.Println(`Attempting to retrieve `, id)
		input := &s3.GetObjectInput{
			Bucket: aws.String(os.Getenv(`BUCKET`)),
			// Key needs to come from a request parameter
			Key: aws.String(`videos/` + id),
		}
		result, err := s3svc.GetObject(context.Background(), input)
		slog.Info(
			"s3svc.GetObject",
			`bucket`, *input.Bucket,
			`key`, *input.Key,
			`error`, err,
		)
		if err != nil {
			w.WriteHeader(404)
			log.Printf("%s not found.", *input.Key)
			return
		}
		defer result.Body.Close()

		io.Copy(w, result.Body)
	})

	slog.Info(`s3svc.Main started`)
	http.ListenAndServe(":80", mux)
}
