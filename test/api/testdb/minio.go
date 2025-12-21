//go:build api

package testdb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// MinIOAccessKey is the default access key for test MinIO.
	MinIOAccessKey = "minioadmin"
	// MinIOSecretKey is the default secret key for test MinIO.
	MinIOSecretKey = "minioadmin"
	// MinIOBucket is the default bucket name for tests.
	MinIOBucket = "test-bucket"
)

// MinIOContainer wraps a MinIO testcontainer for API tests.
type MinIOContainer struct {
	Container testcontainers.Container
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Client    *s3.Client
}

// SetupMinIO starts a MinIO testcontainer.
func SetupMinIO(ctx context.Context) (*MinIOContainer, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     MinIOAccessKey,
			"MINIO_ROOT_PASSWORD": MinIOSecretKey,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	port, err := container.MappedPort(ctx, "9000")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	endpoint := host + ":" + port.Port()

	// Create S3 client for MinIO
	client, err := createMinIOClient(ctx, endpoint)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	mc := &MinIOContainer{
		Container: container,
		Endpoint:  endpoint,
		AccessKey: MinIOAccessKey,
		SecretKey: MinIOSecretKey,
		Bucket:    MinIOBucket,
		Client:    client,
	}

	// Create the test bucket
	if err := mc.createBucket(ctx); err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	return mc, nil
}

// createMinIOClient creates an S3 client configured for MinIO.
func createMinIOClient(ctx context.Context, endpoint string) (*s3.Client, error) {
	endpointURL := "http://" + endpoint

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpointURL,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(MinIOAccessKey, MinIOSecretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return client, nil
}

// createBucket creates the test bucket in MinIO.
func (mc *MinIOContainer) createBucket(ctx context.Context) error {
	_, err := mc.Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(mc.Bucket),
	})
	return err
}

// Cleanup terminates the MinIO container.
func (mc *MinIOContainer) Cleanup(ctx context.Context) error {
	if mc.Container != nil {
		return mc.Container.Terminate(ctx)
	}
	return nil
}

// ClearBucket removes all objects from the bucket.
// Uses batch deletion (up to 1000 objects per request) for efficiency.
// Handles pagination for buckets with more than 1000 objects.
func (mc *MinIOContainer) ClearBucket(ctx context.Context) error {
	var continuationToken *string

	for {
		// List objects with pagination support
		listOutput, err := mc.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(mc.Bucket),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return err
		}

		// Skip if no objects in this page
		if len(listOutput.Contents) == 0 {
			if listOutput.IsTruncated == nil || !*listOutput.IsTruncated {
				break
			}
			continuationToken = listOutput.NextContinuationToken
			continue
		}

		// Build batch delete request (up to 1000 objects per request)
		objectIds := make([]types.ObjectIdentifier, 0, len(listOutput.Contents))
		for _, obj := range listOutput.Contents {
			objectIds = append(objectIds, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		// Batch delete all objects in this page
		_, err = mc.Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(mc.Bucket),
			Delete: &types.Delete{
				Objects: objectIds,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return err
		}

		// Check if there are more objects to list
		if listOutput.IsTruncated == nil || !*listOutput.IsTruncated {
			break
		}
		continuationToken = listOutput.NextContinuationToken
	}

	return nil
}

// ObjectExists checks if an object exists in the bucket.
func (mc *MinIOContainer) ObjectExists(ctx context.Context, key string) bool {
	_, err := mc.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(mc.Bucket),
		Key:    aws.String(key),
	})
	return err == nil
}
