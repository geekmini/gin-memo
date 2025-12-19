// Package storage provides object storage functionality using S3-compatible services.
package storage

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps the S3 client for generating pre-signed URLs.
type S3Client struct {
	client       *s3.Client
	presignClient *s3.PresignClient
	bucket       string
}

// NewS3Client creates a new S3 client configured for the given endpoint.
func NewS3Client(endpoint, accessKey, secretKey, bucket string, useSSL bool) *S3Client {
	// Build the endpoint URL
	protocol := "http"
	if useSSL {
		protocol = "https"
	}
	endpointURL := protocol + "://" + endpoint

	// Create custom resolver for MinIO/S3-compatible endpoints
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpointURL,
			HostnameImmutable: true,
		}, nil
	})

	// Load config with custom credentials and endpoint
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"), // MinIO requires a region
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Fatalf("Failed to load S3 config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for MinIO
	})

	log.Printf("Connected to S3 at %s", endpointURL)

	return &S3Client{
		client:       client,
		presignClient: s3.NewPresignClient(client),
		bucket:       bucket,
	}
}

// GetPresignedURL generates a pre-signed URL for downloading an object.
func (s *S3Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	request, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}

	return request.URL, nil
}
