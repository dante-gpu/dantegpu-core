package api

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dante-gpu/dante-backend/storage-service/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	defaultPresignedURLExpiry = 15 * time.Minute
	maxUploadSize             = 5 * 1024 * 1024 * 1024 // 5 GB, example limit
)

// StorageHandler handles HTTP requests for storage operations.
type StorageHandler struct {
	storageClient storage.ObjectStorage
	logger        *zap.Logger
}

// NewStorageHandler creates a new StorageHandler.
func NewStorageHandler(storageClient storage.ObjectStorage, logger *zap.Logger) *StorageHandler {
	return &StorageHandler{
		storageClient: storageClient,
		logger:        logger.Named("storage_handler"),
	}
}

// RegisterRoutes registers the storage API routes with the given router.
func (h *StorageHandler) RegisterRoutes(r chi.Router) {
	// Bucket operations (less common, typically admin-level or for specific use cases)
	r.Post("/buckets/{bucketName}", h.ensureBucketHandler) // Ensure/Create bucket

	// Object operations (common)
	r.Get("/objects/{bucketName}/*", h.downloadObjectHandler)  // Download (GET with wildcard for object key)
	r.Put("/objects/{bucketName}/*", h.uploadObjectHandler)    // Upload/Replace (PUT with wildcard for object key)
	r.Delete("/objects/{bucketName}/*", h.deleteObjectHandler) // Delete (DELETE with wildcard for object key)
	r.Head("/objects/{bucketName}/*", h.getObjectInfoHandler)  // Get Info (HEAD with wildcard for object key)
	r.Get("/objects/{bucketName}/list", h.listObjectsHandler)  // List objects in a bucket (or with prefix if query param used)

	// Presigned URL generation
	r.Post("/presigned-url/{bucketName}/*", h.generatePresignedURLHandler)

	// Convenience route for default bucket - uses configured default bucket
	r.Get("/objects/*", h.downloadObjectFromDefaultBucketHandler)
	r.Put("/objects/*", h.uploadObjectToDefaultBucketHandler)
	r.Delete("/objects/*", h.deleteObjectFromDefaultBucketHandler)
	r.Head("/objects/*", h.getObjectInfoFromDefaultBucketHandler)
	r.Get("/objects/list", h.listObjectsInDefaultBucketHandler)
	r.Post("/presigned-url/*", h.generatePresignedURLForDefaultBucketHandler)

	h.logger.Info("Storage service routes registered")
}

// respondWithError sends a JSON error response.
func (h *StorageHandler) respondWithError(w http.ResponseWriter, _ *http.Request, code int, message string, err error) {
	logFields := []zap.Field{
		zap.Int("status_code", code),
		zap.String("error_message", message),
	}
	if err != nil {
		logFields = append(logFields, zap.Error(err))
	}
	h.logger.Error("HTTP handler error", logFields...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// respondWithJSON sends a JSON success response.
func (h *StorageHandler) respondWithJSON(w http.ResponseWriter, _ *http.Request, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			h.logger.Error("Failed to encode JSON response", zap.Error(err), zap.Any("payload", payload))
			// Don't try to write again if headers already sent, but log it.
		}
	}
}

// ensureBucketHandler handles requests to ensure a bucket exists (creates if not).
func (h *StorageHandler) ensureBucketHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	if bucketName == "" {
		h.respondWithError(w, r, http.StatusBadRequest, "Bucket name is required", nil)
		return
	}
	// Region could be part of the request body if dynamic region creation is needed, or use a default.
	// For simplicity, we assume a default region is configured with the MinIO client if needed.
	if err := h.storageClient.EnsureBucket(r.Context(), bucketName, ""); err != nil {
		h.respondWithError(w, r, http.StatusInternalServerError, "Failed to ensure bucket", err)
		return
	}
	h.respondWithJSON(w, r, http.StatusCreated, map[string]string{"message": "Bucket ensured successfully", "bucket": bucketName})
}

// getObjectKeyFromPath extracts the object key from the Chi URL parameter pattern `/*`.
func getObjectKeyFromPath(r *http.Request) string {
	// chi.URLParam(r, "*") should give the part of the path matched by the wildcard
	return chi.URLParam(r, "*")
}

// uploadObjectHandler handles object uploads.
func (h *StorageHandler) uploadObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	objectKey := getObjectKeyFromPath(r)

	if objectKey == "" {
		h.respondWithError(w, r, http.StatusBadRequest, "Object key is required in the path", nil)
		return
	}

	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	defer r.Body.Close()

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		// Attempt to guess content type from extension if not provided
		ext := filepath.Ext(objectKey)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream" // Default
		}
	}

	// The Content-Length header might not be accurate or present for all clients/proxies,
	// especially with chunked encoding. MinIO's PutObject can handle size -1 (unknown).
	// However, if Content-Length is available and trustworthy, it can be used.
	contentLengthStr := r.Header.Get("Content-Length")
	size, _ := strconv.ParseInt(contentLengthStr, 10, 64) // Ignore error, size will be 0 if invalid or -1 for unknown

	info, err := h.storageClient.Upload(r.Context(), bucketName, objectKey, r.Body, size, contentType)
	if err != nil {
		if strings.Contains(err.Error(), "http.MaxBytesReader") {
			h.respondWithError(w, r, http.StatusRequestEntityTooLarge, "Upload exceeds maximum allowed size", err)
		} else {
			h.respondWithError(w, r, http.StatusInternalServerError, "Failed to upload object", err)
		}
		return
	}
	h.respondWithJSON(w, r, http.StatusCreated, info)
}

// downloadObjectHandler handles object downloads.
func (h *StorageHandler) downloadObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	objectKey := getObjectKeyFromPath(r)

	if objectKey == "" {
		h.respondWithError(w, r, http.StatusBadRequest, "Object key is required in the path", nil)
		return
	}

	objStream, info, err := h.storageClient.Download(r.Context(), bucketName, objectKey)
	if err != nil {
		// TODO: Differentiate between "not found" and other errors for status code
		h.respondWithError(w, r, http.StatusInternalServerError, "Failed to download object", err)
		return
	}
	defer objStream.Close()

	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("ETag", info.ETag)
	w.Header().Set("Last-Modified", info.LastModified.Format(http.TimeFormat))
	// Consider adding "Content-Disposition" for filename on download
	// w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(objectKey)))

	if _, err := io.Copy(w, objStream); err != nil {
		// Error already logged by this point if it's a client-side issue (e.g., connection closed)
		// For server-side issues during copy, log here.
		h.logger.Error("Failed to stream object to client", zap.Error(err), zap.String("key", objectKey))
		// Cannot send error response if headers already sent / body partially written
		return
	}
}

// deleteObjectHandler handles object deletions.
func (h *StorageHandler) deleteObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	objectKey := getObjectKeyFromPath(r)

	if objectKey == "" {
		h.respondWithError(w, r, http.StatusBadRequest, "Object key is required in the path", nil)
		return
	}

	if err := h.storageClient.Delete(r.Context(), bucketName, objectKey); err != nil {
		h.respondWithError(w, r, http.StatusInternalServerError, "Failed to delete object", err)
		return
	}
	h.respondWithJSON(w, r, http.StatusNoContent, nil)
}

// getObjectInfoHandler handles requests for object metadata (HEAD requests).
func (h *StorageHandler) getObjectInfoHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	objectKey := getObjectKeyFromPath(r)

	if objectKey == "" {
		h.respondWithError(w, r, http.StatusBadRequest, "Object key is required in the path", nil)
		return
	}

	info, err := h.storageClient.GetObjectInfo(r.Context(), bucketName, objectKey)
	if err != nil {
		// TODO: Differentiate between "not found" (404) and other errors (500)
		h.respondWithError(w, r, http.StatusNotFound, "Object not found or failed to get info", err)
		return
	}

	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("ETag", info.ETag)
	w.Header().Set("Last-Modified", info.LastModified.Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
}

// listObjectsHandler handles requests to list objects in a bucket.
func (h *StorageHandler) listObjectsHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	prefix := r.URL.Query().Get("prefix")
	recursiveStr := r.URL.Query().Get("recursive")
	recursive := true // Default to recursive
	if recursiveStr == "false" {
		recursive = false
	}

	objects, err := h.storageClient.ListObjects(r.Context(), bucketName, prefix, recursive)
	if err != nil {
		h.respondWithError(w, r, http.StatusInternalServerError, "Failed to list objects", err)
		return
	}
	h.respondWithJSON(w, r, http.StatusOK, objects)
}

// generatePresignedURLHandler handles requests to generate presigned URLs.
func (h *StorageHandler) generatePresignedURLHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	objectKey := getObjectKeyFromPath(r)

	if objectKey == "" {
		h.respondWithError(w, r, http.StatusBadRequest, "Object key is required in the path", nil)
		return
	}

	var reqBody struct {
		Method string `json:"method"` // "GET" or "PUT"
		Expiry string `json:"expiry"` // Duration string like "15m", "1h"
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.respondWithError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	defer r.Body.Close()

	if reqBody.Method != "GET" && reqBody.Method != "PUT" {
		h.respondWithError(w, r, http.StatusBadRequest, "Invalid method for presigned URL. Must be GET or PUT.", nil)
		return
	}

	expiryDuration := defaultPresignedURLExpiry
	if reqBody.Expiry != "" {
		parsedExpiry, err := time.ParseDuration(reqBody.Expiry)
		if err != nil {
			h.respondWithError(w, r, http.StatusBadRequest, "Invalid expiry duration format", err)
			return
		}
		if parsedExpiry > 0 {
			expiryDuration = parsedExpiry
		}
	}

	presignedURL, err := h.storageClient.GetPresignedURL(r.Context(), bucketName, objectKey, reqBody.Method, expiryDuration)
	if err != nil {
		h.respondWithError(w, r, http.StatusInternalServerError, "Failed to generate presigned URL", err)
		return
	}

	h.respondWithJSON(w, r, http.StatusOK, map[string]string{"url": presignedURL, "method": reqBody.Method, "key": objectKey, "bucket": bucketName})
}

// --- Default Bucket Handler Wrappers --- //

func (h *StorageHandler) uploadObjectToDefaultBucketHandler(w http.ResponseWriter, r *http.Request) {
	// Modify request to set bucketName to empty string, so storage client uses default
	// This is a bit of a hack with chi context, ideally the route matching would directly allow this.
	// A cleaner way might be to have separate handler functions or a middleware that sets default bucket in context.
	ctx := chi.RouteContext(r.Context())
	ctx.URLParams.Keys[0] = "bucketName"                 // Assuming 'bucketName' is the first param for the generic route
	ctx.URLParams.Values[0] = ""                         // Set to empty, which MinioClient will use as default
	h.uploadObjectHandler(w, r.WithContext(r.Context())) // Pass new context if needed
}

func (h *StorageHandler) downloadObjectFromDefaultBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := chi.RouteContext(r.Context())
	ctx.URLParams.Keys[0] = "bucketName"
	ctx.URLParams.Values[0] = ""
	h.downloadObjectHandler(w, r)
}

func (h *StorageHandler) deleteObjectFromDefaultBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := chi.RouteContext(r.Context())
	ctx.URLParams.Keys[0] = "bucketName"
	ctx.URLParams.Values[0] = ""
	h.deleteObjectHandler(w, r)
}

func (h *StorageHandler) getObjectInfoFromDefaultBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := chi.RouteContext(r.Context())
	ctx.URLParams.Keys[0] = "bucketName"
	ctx.URLParams.Values[0] = ""
	h.getObjectInfoHandler(w, r)
}

func (h *StorageHandler) listObjectsInDefaultBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := chi.RouteContext(r.Context())
	ctx.URLParams.Keys[0] = "bucketName"
	ctx.URLParams.Values[0] = ""
	h.listObjectsHandler(w, r)
}

func (h *StorageHandler) generatePresignedURLForDefaultBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := chi.RouteContext(r.Context())
	ctx.URLParams.Keys[0] = "bucketName"
	ctx.URLParams.Values[0] = ""
	h.generatePresignedURLHandler(w, r)
}
