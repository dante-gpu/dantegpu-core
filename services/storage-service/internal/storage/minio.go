package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/dante-gpu/dante-backend/storage-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// MinioClient wraps the MinIO client and implements the ObjectStorage interface.
type MinioClient struct {
	client        *minio.Client
	logger        *zap.Logger
	config        config.MinioConfig
	defaultBucket string
}

// NewMinioClient creates and returns a new MinIO client.
func NewMinioClient(cfg config.MinioConfig, logger *zap.Logger) (*MinioClient, error) {
	logger.Info("Initializing MinIO client",
		zap.String("endpoint", cfg.Endpoint),
		zap.Bool("useSSL", cfg.UseSSL),
		zap.String("region", cfg.Region),
		zap.String("defaultBucket", cfg.DefaultBucket),
	)

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region, // Optional: Set region if applicable
	})
	if err != nil {
		logger.Error("Failed to create MinIO client", zap.Error(err))
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Optional: Ping MinIO server to check connectivity
	// Using a short timeout for this check.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.ListBuckets(ctx) // A simple operation to test connection and credentials
	if err != nil {
		logger.Error("Failed to connect to MinIO server or authenticate", zap.Error(err))
		// Depending on policy, you might not want to return an error here,
		// but rather allow retries or handle it at a higher level.
		// For now, we return an error.
		return nil, fmt.Errorf("failed to connect/authenticate with MinIO: %w", err)
	}
	logger.Info("Successfully connected to MinIO server")

	return &MinioClient{
		client:        client,
		logger:        logger.Named("minio_storage"),
		config:        cfg,
		defaultBucket: cfg.DefaultBucket,
	}, nil
}

// EnsureBucket creates a bucket if it does not already exist.
func (mc *MinioClient) EnsureBucket(ctx context.Context, bucketName string, region string) error {
	mc.logger.Debug("Ensuring bucket exists", zap.String("bucket", bucketName), zap.String("region", region))
	exists, err := mc.client.BucketExists(ctx, bucketName)
	if err != nil {
		mc.logger.Error("Failed to check if bucket exists", zap.String("bucket", bucketName), zap.Error(err))
		return fmt.Errorf("failed to check for bucket %s: %w", bucketName, err)
	}
	if !exists {
		mc.logger.Info("Bucket does not exist, creating it", zap.String("bucket", bucketName), zap.String("region", region))
		err = mc.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: region, ObjectLocking: false})
		if err != nil {
			mc.logger.Error("Failed to create bucket", zap.String("bucket", bucketName), zap.Error(err))
			return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
		mc.logger.Info("Bucket created successfully", zap.String("bucket", bucketName))
	} else {
		mc.logger.Debug("Bucket already exists", zap.String("bucket", bucketName))
	}
	return nil
}

// getTargetBucket determines the bucket to use, defaulting to the client's default bucket if none is provided.
func (mc *MinioClient) getTargetBucket(bucketName string) string {
	if bucketName == "" {
		return mc.defaultBucket
	}
	return bucketName
}

// Upload uploads an object to the specified bucket.
// If bucketName is empty, the default bucket is used.
func (mc *MinioClient) Upload(ctx context.Context, bucketName, objectKey string, reader io.Reader, size int64, contentType string) (*ObjectInfo, error) {
	targetBucket := mc.getTargetBucket(bucketName)
	if targetBucket == "" {
		return nil, fmt.Errorf("bucket name is not specified and no default bucket is configured")
	}
	mc.logger.Debug("Uploading object",
		zap.String("bucket", targetBucket),
		zap.String("key", objectKey),
		zap.Int64("size", size),
		zap.String("contentType", contentType),
	)

	opts := minio.PutObjectOptions{
		ContentType: contentType,
		// TODO: Add other options like UserMetadata, Progress, etc. if needed
	}

	uploadInfo, err := mc.client.PutObject(ctx, targetBucket, objectKey, reader, size, opts)
	if err != nil {
		mc.logger.Error("Failed to upload object", zap.String("bucket", targetBucket), zap.String("key", objectKey), zap.Error(err))
		return nil, fmt.Errorf("failed to upload to %s/%s: %w", targetBucket, objectKey, err)
	}

	mc.logger.Info("Object uploaded successfully",
		zap.String("bucket", uploadInfo.Bucket),
		zap.String("key", uploadInfo.Key),
		zap.String("etag", uploadInfo.ETag),
		zap.Int64("size", uploadInfo.Size),
	)

	return &ObjectInfo{
		Key:          uploadInfo.Key,
		Size:         uploadInfo.Size,
		ETag:         uploadInfo.ETag,
		ContentType:  contentType,      // Minio PutObjectInfo doesn't directly return this, so we use the input.
		LastModified: time.Now().UTC(), // PutObjectInfo doesn't return LastModified. Consider GetObjectInfo after put.
	}, nil
}

// Download downloads an object from the specified bucket.
// If bucketName is empty, the default bucket is used.
func (mc *MinioClient) Download(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, *ObjectInfo, error) {
	targetBucket := mc.getTargetBucket(bucketName)
	if targetBucket == "" {
		return nil, nil, fmt.Errorf("bucket name is not specified and no default bucket is configured")
	}
	mc.logger.Debug("Downloading object", zap.String("bucket", targetBucket), zap.String("key", objectKey))

	obj, err := mc.client.GetObject(ctx, targetBucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		mc.logger.Error("Failed to get object", zap.String("bucket", targetBucket), zap.String("key", objectKey), zap.Error(err))
		return nil, nil, fmt.Errorf("failed to get object %s/%s: %w", targetBucket, objectKey, err)
	}

	stat, err := obj.Stat()
	if err != nil {
		mc.logger.Error("Failed to get object stats after GetObject", zap.String("bucket", targetBucket), zap.String("key", objectKey), zap.Error(err))
		// Close the object stream if stat fails, as the caller won't be able to.
		obj.Close()
		return nil, nil, fmt.Errorf("failed to get object stats for %s/%s: %w", targetBucket, objectKey, err)
	}

	mc.logger.Info("Object ready for download",
		zap.String("bucket", targetBucket),
		zap.String("key", stat.Key),
		zap.Int64("size", stat.Size),
		zap.String("contentType", stat.ContentType),
	)

	return obj, &ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		LastModified: stat.LastModified,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
	}, nil
}

// Delete deletes an object from the specified bucket.
// If bucketName is empty, the default bucket is used.
func (mc *MinioClient) Delete(ctx context.Context, bucketName, objectKey string) error {
	targetBucket := mc.getTargetBucket(bucketName)
	if targetBucket == "" {
		return fmt.Errorf("bucket name is not specified and no default bucket is configured")
	}
	mc.logger.Debug("Deleting object", zap.String("bucket", targetBucket), zap.String("key", objectKey))

	err := mc.client.RemoveObject(ctx, targetBucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		mc.logger.Error("Failed to delete object", zap.String("bucket", targetBucket), zap.String("key", objectKey), zap.Error(err))
		return fmt.Errorf("failed to delete object %s/%s: %w", targetBucket, objectKey, err)
	}

	mc.logger.Info("Object deleted successfully", zap.String("bucket", targetBucket), zap.String("key", objectKey))
	return nil
}

// ListObjects lists objects in the specified bucket, optionally filtered by prefix.
// If bucketName is empty, the default bucket is used.
func (mc *MinioClient) ListObjects(ctx context.Context, bucketName, prefix string, recursive bool) ([]*ObjectInfo, error) {
	targetBucket := mc.getTargetBucket(bucketName)
	if targetBucket == "" {
		return nil, fmt.Errorf("bucket name is not specified and no default bucket is configured")
	}
	mc.logger.Debug("Listing objects",
		zap.String("bucket", targetBucket),
		zap.String("prefix", prefix),
		zap.Bool("recursive", recursive),
	)

	var objects []*ObjectInfo
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	}

	objectCh := mc.client.ListObjects(ctx, targetBucket, opts)
	for object := range objectCh {
		if object.Err != nil {
			mc.logger.Error("Error listing objects", zap.String("bucket", targetBucket), zap.Error(object.Err))
			return nil, fmt.Errorf("error during object listing in %s: %w", targetBucket, object.Err)
		}
		objects = append(objects, &ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ContentType:  object.ContentType, // ContentType might not always be populated by ListObjects
			ETag:         object.ETag,
		})
	}

	mc.logger.Info("Objects listed successfully", zap.String("bucket", targetBucket), zap.Int("count", len(objects)))
	return objects, nil
}

// GetObjectInfo retrieves metadata for a specific object.
// If bucketName is empty, the default bucket is used.
func (mc *MinioClient) GetObjectInfo(ctx context.Context, bucketName, objectKey string) (*ObjectInfo, error) {
	targetBucket := mc.getTargetBucket(bucketName)
	if targetBucket == "" {
		return nil, fmt.Errorf("bucket name is not specified and no default bucket is configured")
	}
	mc.logger.Debug("Getting object info", zap.String("bucket", targetBucket), zap.String("key", objectKey))

	stat, err := mc.client.StatObject(ctx, targetBucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		mc.logger.Error("Failed to get object info", zap.String("bucket", targetBucket), zap.String("key", objectKey), zap.Error(err))
		// Handle MinIO's specific error for "not found" if necessary, or let it propagate.
		// Example:
		// if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		// 	 return nil, ErrObjectNotFound // Define your own error for this
		// }
		return nil, fmt.Errorf("failed to get object info for %s/%s: %w", targetBucket, objectKey, err)
	}

	mc.logger.Info("Object info retrieved successfully",
		zap.String("bucket", targetBucket),
		zap.String("key", stat.Key),
		zap.Int64("size", stat.Size),
	)
	return &ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		LastModified: stat.LastModified,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
	}, nil
}

// GetPresignedURL generates a presigned URL for an object.
// If bucketName is empty, the default bucket is used.
// For GET operations (method="GET"), expiry indicates how long the URL is valid.
// For PUT operations (method="PUT"), expiry dictates the time window for the upload.
func (mc *MinioClient) GetPresignedURL(ctx context.Context, bucketName, objectKey, method string, expiry time.Duration) (string, error) {
	targetBucket := mc.getTargetBucket(bucketName)
	if targetBucket == "" {
		return "", fmt.Errorf("bucket name is not specified and no default bucket is configured")
	}
	mc.logger.Debug("Generating presigned URL",
		zap.String("bucket", targetBucket),
		zap.String("key", objectKey),
		zap.String("method", method),
		zap.Duration("expiry", expiry),
	)

	if expiry <= 0 {
		expiry = 7 * 24 * time.Hour // Default to 7 days if not specified or invalid
		mc.logger.Warn("Expiry for presigned URL was zero or negative, defaulted to 7 days", zap.String("key", objectKey))
	}

	reqParams := make(url.Values)
	// Example: reqParams.Set("response-content-disposition", "attachment; filename=\"your-filename.txt\"")

	var presignedURL *url.URL
	var err error

	if method == "GET" {
		presignedURL, err = mc.client.PresignedGetObject(ctx, targetBucket, objectKey, expiry, reqParams)
	} else if method == "PUT" {
		presignedURL, err = mc.client.PresignedPutObject(ctx, targetBucket, objectKey, expiry)
		// Note: For PresignedPutObject, MinIO Go SDK v7 doesn't directly accept reqParams in the same way PresignedGetObject does.
		// If you need to set headers for the actual PUT operation that uses this URL (e.g. ContentType),
		// they must be included by the client making the PUT request.
		// The presigned URL grants permission, the client provides the data and metadata.
	} else {
		return "", fmt.Errorf("unsupported HTTP method for presigned URL: %s", method)
	}

	if err != nil {
		mc.logger.Error("Failed to generate presigned URL",
			zap.String("bucket", targetBucket),
			zap.String("key", objectKey),
			zap.String("method", method),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to generate presigned URL for %s %s/%s: %w", method, targetBucket, objectKey, err)
	}

	mc.logger.Info("Presigned URL generated successfully",
		zap.String("bucket", targetBucket),
		zap.String("key", objectKey),
		zap.String("method", method),
	)
	return presignedURL.String(), nil
}
