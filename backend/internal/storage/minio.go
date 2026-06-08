package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"sim-scenario-platform/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStore struct {
	client *minio.Client
	bucket string
}

func NewMinioStore(cfg *config.Config) (*MinioStore, error) {
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create minio client: %w", err)
	}
	return &MinioStore{client: client, bucket: cfg.MinioBucket}, nil
}

func (s *MinioStore) InitBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

func (s *MinioStore) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *MinioStore) DownloadFile(ctx context.Context, objectName string) (*minio.Object, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *MinioStore) GetPresignedURL(ctx context.Context, objectName string) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "inline")
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, 1*time.Hour, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *MinioStore) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	chanObj := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	for obj := range chanObj {
		if obj.Err != nil {
			return nil, obj.Err
		}
		files = append(files, obj.Key)
	}
	return files, nil
}

func (s *MinioStore) DeleteFile(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}
