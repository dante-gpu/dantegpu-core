package storage

import (
	"context"
	"io"
	"time"
)

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ContentType  string    `json:"content_type"`
	ETag         string    `json:"etag"`
}

// ObjectStorage defines the interface for interacting with an object storage backend.
type ObjectStorage interface {
	// Upload uploads a file to the specified bucket with the given key.
	// reader is the content of the file, size is the total size, and contentType is the MIME type.
	Upload(ctx context.Context, bucketName, key string, reader io.Reader, size int64, contentType string) (*ObjectInfo, error)

	// Download retrieves a file from the specified bucket with the given key.
	// It returns an io.ReadCloser for the file content and its metadata.
	Download(ctx context.Context, bucketName, key string) (io.ReadCloser, *ObjectInfo, error)

	// Delete removes a file from the specified bucket with the given key.
	Delete(ctx context.Context, bucketName, key string) error

	// ListObjects lists objects in the specified bucket, optionally filtered by a prefix.
	ListObjects(ctx context.Context, bucketName, prefix string, recursive bool) ([]*ObjectInfo, error)

	// GetObjectInfo retrieves metadata for a specific object.
	GetObjectInfo(ctx context.Context, bucketName, key string) (*ObjectInfo, error)

	// EnsureBucket checks if a bucket exists, and creates it if it doesn't.
	EnsureBucket(ctx context.Context, bucketName string, region string) error

	// GetPresignedURL generates a presigned URL for an object, either for uploading (PUT) or downloading (GET).
	// expiry is the duration for which the URL will be valid.
	GetPresignedURL(ctx context.Context, bucketName, key string, method string, expiry time.Duration) (string, error)
}
