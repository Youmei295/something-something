// Package storage implements ports.Storage against any S3-compatible backend.
// The same adapter targets MinIO locally and AWS S3 in production by changing
// configuration only.
package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Options configures the S3 adapter.
type Options struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	Region         string
	UseSSL         bool
}

// S3 implements ports.Storage.
type S3 struct {
	client  *minio.Client // used for server-side operations
	presign *minio.Client // used only to sign URLs (public host)
	bucket  string
	region  string
}

// New builds an S3 adapter. When PublicEndpoint is set, presigned URLs are
// signed against it so browsers can reach storage behind a different host.
func New(ctx context.Context, opts Options) (*S3, error) {
	client, err := minio.New(opts.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure: opts.UseSSL,
		Region: opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: new client: %w", err)
	}

	presign := client
	if opts.PublicEndpoint != "" {
		presign, err = minio.New(opts.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
			Secure: opts.UseSSL,
			Region: opts.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("storage: new presign client: %w", err)
		}
	}

	_ = ctx
	return &S3{client: client, presign: presign, bucket: opts.Bucket, region: opts.Region}, nil
}

// EnsureBucket creates the bucket if it does not exist. Safe to call on startup.
func (s *S3) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("storage: bucket exists: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region}); err != nil {
		return fmt.Errorf("storage: make bucket: %w", err)
	}
	return nil
}

// PresignPut returns a URL the browser can PUT a file to directly.
func (s *S3) PresignPut(ctx context.Context, key, _ string, ttl time.Duration) (string, error) {
	u, err := s.presign.PresignedPutObject(ctx, s.bucket, key, ttl)
	if err != nil {
		return "", fmt.Errorf("storage: presign put: %w", err)
	}
	return u.String(), nil
}

// PresignGet returns a URL that downloads the object with a friendly filename.
func (s *S3) PresignGet(ctx context.Context, key, downloadFilename, contentType string, ttl time.Duration) (string, error) {
	params := url.Values{}
	if downloadFilename != "" {
		params.Set("response-content-disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(downloadFilename)))
	}
	if contentType != "" {
		params.Set("response-content-type", contentType)
	}
	u, err := s.presign.PresignedGetObject(ctx, s.bucket, key, ttl, params)
	if err != nil {
		return "", fmt.Errorf("storage: presign get: %w", err)
	}
	return u.String(), nil
}

// Stat returns object metadata, used to verify an upload actually happened.
func (s *S3) Stat(ctx context.Context, key string) (ports.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ports.ObjectInfo{}, fmt.Errorf("storage: stat: %w", err)
	}
	return ports.ObjectInfo{
		Size:        info.Size,
		ContentType: info.ContentType,
		ETag:        info.ETag,
	}, nil
}

// Remove deletes an object, ignoring not-found errors.
func (s *S3) Remove(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage: remove: %w", err)
	}
	return nil
}

func sanitizeFilename(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r == '"' || r == '\\' || r == '\r' || r == '\n':
			continue
		default:
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "download"
	}
	return string(out)
}

var _ ports.Storage = (*S3)(nil)
