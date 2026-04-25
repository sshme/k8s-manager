package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	defaultS3Endpoint = "localhost:9000"
	defaultS3Bucket   = "k8s-manager-artifacts"
)

// S3Storage implements artifact storage using an S3-compatible object store.
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3StorageFromEnv creates S3 storage from S3_* environment variables.
func NewS3StorageFromEnv(ctx context.Context) (*S3Storage, error) {
	endpoint := getEnv("S3_ENDPOINT", defaultS3Endpoint)
	accessKey := getEnv("S3_ACCESS_KEY", "minioadmin")
	secretKey := getEnv("S3_SECRET_KEY", "minioadmin")
	bucket := getEnv("S3_BUCKET", defaultS3Bucket)
	useSSL := getEnvBool("S3_USE_SSL", false)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	if err := ensureBucket(ctx, client, bucket); err != nil {
		return nil, err
	}

	return &S3Storage{client: client, bucket: bucket}, nil
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		exists, err := client.BucketExists(ctx, bucket)
		if err == nil && exists {
			return nil
		}
		if err == nil {
			if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				lastErr = err
			} else {
				return nil
			}
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return fmt.Errorf("ensure s3 bucket: %w", lastErr)
}

// Save saves an artifact object and returns its object key and checksum.
func (s *S3Storage) Save(pluginID, releaseID int64, osName, arch, filename string, reader io.Reader) (string, string, int64, error) {
	if filename == "" {
		filename = "artifact.zip"
	}
	filename = filepath.Base(filename)
	if filename == "." || filename == string(filepath.Separator) {
		filename = "artifact.zip"
	}

	key := fmt.Sprintf("plugins/%d/releases/%d/%s/%s/%s", pluginID, releaseID, osName, arch, filename)

	var data bytes.Buffer
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(&data, hash), reader)
	if err != nil {
		return "", "", 0, fmt.Errorf("buffer artifact for s3: %w", err)
	}

	_, err = s.client.PutObject(
		context.Background(),
		s.bucket,
		key,
		bytes.NewReader(data.Bytes()),
		size,
		minio.PutObjectOptions{
			ContentType: "application/zip",
		},
	)
	if err != nil {
		return "", "", 0, fmt.Errorf("put s3 object: %w", err)
	}

	return key, hex.EncodeToString(hash.Sum(nil)), size, nil
}

// Get retrieves an artifact object.
func (s *S3Storage) Get(storagePath string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(context.Background(), s.bucket, storagePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get s3 object: %w", err)
	}

	return object, nil
}

// Delete deletes an artifact object.
func (s *S3Storage) Delete(storagePath string) error {
	err := s.client.RemoveObject(context.Background(), s.bucket, storagePath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("delete s3 object: %w", err)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
