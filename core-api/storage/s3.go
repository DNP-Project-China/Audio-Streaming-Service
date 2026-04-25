package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/fx"
)

// DI for S3 storage
var Module = fx.Options(
	fx.Provide(NewS3Storage),
)

type S3Storage struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
	cfg     *server.Config
}

type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified *time.Time
}

// DI constructor for S3Storage
func NewS3Storage(cfg *server.Config) (*S3Storage, error) {
	store := &S3Storage{
		bucket: cfg.S3Bucket,
		cfg:    cfg,
	}

	awsConfig, err := awscfg.LoadDefaultConfig(
		context.Background(),
		awscfg.WithRegion(cfg.S3Region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
		awscfg.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awscfg.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(cfg.S3Endpoint)
	})

	store.client = client
	store.presign = s3.NewPresignClient(client)
	return store, nil
}

// Getter for bucket name
func (s *S3Storage) Bucket() string {
	if s == nil {
		return ""
	}
	return s.bucket
}

// Convenience method to put an object into S3
func (s *S3Storage) PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(strings.TrimPrefix(key, "/")),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}

	return nil
}

// Convenience method to check if an object exists in S3
func (s *S3Storage) HeadObject(ctx context.Context, key string) error {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(strings.TrimPrefix(key, "/")),
	})
	if err != nil {
		return fmt.Errorf("head object %q: %w", key, err)
	}

	return nil
}

// Convenience method to get an object from S3
func (s *S3Storage) GetObject(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(strings.TrimPrefix(key, "/")),
	})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read object %q body: %w", key, err)
	}

	return data, nil
}

// List objects in S3 with a given prefix
func (s *S3Storage) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]ObjectInfo, error) {
	// Default maxKeys to 100 if not set or invalid
	if maxKeys <= 0 {
		maxKeys = 100
	}

	out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		Prefix:  aws.String(strings.TrimPrefix(prefix, "/")),
		MaxKeys: aws.Int32(maxKeys),
	})
	if err != nil {
		return nil, fmt.Errorf("list objects %q: %w", prefix, err)
	}

	// Convert AWS SDK's object list to ObjectInfo slice
	items := make([]ObjectInfo, 0, len(out.Contents))
	for _, obj := range out.Contents {
		items = append(items, ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: obj.LastModified,
		})
	}

	return items, nil
}

// Convenience method to delete an object from S3
func (s *S3Storage) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(strings.TrimPrefix(key, "/")),
	})
	if err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}

	return nil
}

// Generate a presigned URL for getting an object from S3
func (s *S3Storage) PresignGetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	out, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(strings.TrimPrefix(key, "/")),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", fmt.Errorf("presign get object %q: %w", key, err)
	}

	return out.URL, nil
}

// Generate a presigned URL for putting an object into S3
func (s *S3Storage) PresignPutURL(ctx context.Context, key string, contentType string, expires time.Duration) (string, error) {
	out, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(strings.TrimPrefix(key, "/")),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", fmt.Errorf("presign put object %q: %w", key, err)
	}

	return out.URL, nil
}

// Build a public URL for an object in S3
func (s *S3Storage) PublicObjectURL(key string) (string, error) {
	if s == nil || s.cfg == nil {
		return "", fmt.Errorf("storage config is nil")
	}

	cleanKey := strings.TrimPrefix(path.Clean("/"+key), "/")
	return s.cfg.BuildPublicObjectURL(cleanKey)
}
