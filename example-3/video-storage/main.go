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

func failOnMissingEnvironmentVariable(variableName, failureMessage string) string {
	value, found := os.LookupEnv(variableName)
	if !found {
		log.Fatal(failureMessage)
	}

	return value
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(`middleware`, `method`, r.Method, `host`, r.Host, `url`, r.URL)
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := failOnMissingEnvironmentVariable(`PORT`, `Please specify the port number for the HTTP server with the environment variable PORT.`)
	minioStorageHost := failOnMissingEnvironmentVariable(`MINIO_STORAGE_HOST`, `Please specify the name for the storage host in the variable MINIO_STORAGE_HOST.`)
	minioStoragePort := failOnMissingEnvironmentVariable(`MINIO_STORAGE_PORT`, `Please specify the port number for the storage host with the environment variable MINIO_STORAGE_PORT.`)
	bucketName := failOnMissingEnvironmentVariable("BUCKET", `Please specify the S3 bucket name in the environment variable BUCKET.`)
	minioUser := failOnMissingEnvironmentVariable(`MINIO_ROOT_USER`, `Please specify the S3 user name in the environment variable MINIO_ROOT_USER.`)
	minioPassword := failOnMissingEnvironmentVariable(`MINIO_ROOT_PASSWORD`, `Please specify the S3 password in the environment variable MINIO_ROOT_PASSWORD.`)

	minioEndpoint := fmt.Sprintf(`%s:%s`, minioStorageHost, minioStoragePort)
	awsRegion := "us-east-1"

	credProvider := aws.NewCredentialsCache(
		credentials.NewStaticCredentialsProvider(
			minioUser,
			minioPassword,
			""))

	s3svc := s3.New(s3.Options{
		Credentials:  credProvider,
		Region:       awsRegion,
		BaseEndpoint: &minioEndpoint,
		UsePathStyle: true,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /video", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue(`id`)
		slog.Info(`GET /video`, `id`, id, `from`, bucketName)
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			// Key needs to come from a request parameter
			Key: aws.String(id),
		}
		result, err := s3svc.GetObject(context.Background(), input)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			slog.Error(fmt.Sprintf(`%s`, *input.Key), `bucket`, bucketName, `result`, result, `err`, err)
			return
		}
		defer result.Body.Close()

		io.Copy(w, result.Body)
	})

	slog.Info(`s3svc.Main started`)
	http.ListenAndServe(fmt.Sprint(":", port), loggingMiddleware(mux))
}
