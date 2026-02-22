package audio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var _ FileStore = (*S3FileStore)(nil)

type S3FileStore struct {
	client *s3.Client
	bucket string
}

func NewS3FileStore(endpoint, bucket, accessKey, secretKey string, useSSL bool) *S3FileStore {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	fullEndpoint := fmt.Sprintf("%s://%s", scheme, endpoint)

	client := s3.New(s3.Options{
		BaseEndpoint: &fullEndpoint,
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		UsePathStyle: true,
	})

	return &S3FileStore{
		client: client,
		bucket: bucket,
	}
}

func (s *S3FileStore) Write(path string, data []byte) error {
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &path,
		Body:        bytes.NewReader(data),
		ContentType: aws.String("audio/mpeg"),
	})
	if err != nil {
		return fmt.Errorf("uploading to S3: %w", err)
	}
	return nil
}

func (s *S3FileStore) Read(path string) ([]byte, error) {
	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
	})
	if err != nil {
		return nil, fmt.Errorf("reading from S3: %w", err)
	}
	defer func() { _ = out.Body.Close() }()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("reading S3 object body: %w", err)
	}
	return data, nil
}
