package video

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"feedsystem_video_go/internal/middleware/jwt"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/storage"

	"github.com/gin-gonic/gin"
)

const sessionTTL = 24 * time.Hour

var (
	errChunkCacheUnavailable   = errors.New("chunk upload requires redis")
	errChunkStorageUnavailable = errors.New("chunk upload requires object storage")
)

type ChunkUploadHandler struct {
	cache        *rediscache.Client
	storage      storage.ObjectStorage
	signedURLTTL time.Duration
}

func NewChunkUploadHandler(
	cache *rediscache.Client,
	objectStore storage.ObjectStorage,
	signedURLTTL time.Duration,
) *ChunkUploadHandler {
	return &ChunkUploadHandler{
		cache:        cache,
		storage:      objectStore,
		signedURLTTL: signedURLTTL,
	}
}

func (h *ChunkUploadHandler) available(c *gin.Context) bool {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable.Error()})
		return false
	}
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkStorageUnavailable.Error()})
		return false
	}
	return true
}

func (h *ChunkUploadHandler) sessionKey(uploadID string) string {
	return h.cache.Key("chunk_upload:%s", uploadID)
}

func (h *ChunkUploadHandler) uploadedKey(uploadID string) string {
	return h.cache.Key("chunk_upload:%s:uploaded", uploadID)
}

func (h *ChunkUploadHandler) etagKey(uploadID string) string {
	return h.cache.Key("chunk_upload:%s:etags", uploadID)
}

func (h *ChunkUploadHandler) hashKey(accountID uint, fileHash string) string {
	return h.cache.Key("chunk_upload_hash:%d:%s", accountID, fileHash)
}

func (h *ChunkUploadHandler) getSession(ctx *gin.Context, uploadID string) (*ChunkUploadSession, error) {
	if h.cache == nil {
		return nil, errChunkCacheUnavailable
	}
	b, err := h.cache.GetBytes(ctx.Request.Context(), h.sessionKey(uploadID))
	if err != nil {
		return nil, fmt.Errorf("upload session not found")
	}
	var session ChunkUploadSession
	if err := json.Unmarshal(b, &session); err != nil {
		return nil, fmt.Errorf("invalid session data")
	}
	return &session, nil
}

func (h *ChunkUploadHandler) saveSession(ctx *gin.Context, session *ChunkUploadSession) error {
	if h.cache == nil {
		return errChunkCacheUnavailable
	}
	b, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return h.cache.SetBytes(ctx.Request.Context(), h.sessionKey(session.UploadID), b, sessionTTL)
}

func (h *ChunkUploadHandler) uploadedChunks(ctx context.Context, uploadID string) ([]int, error) {
	values, err := h.cache.SMembers(ctx, h.uploadedKey(uploadID))
	if err != nil {
		return nil, err
	}
	chunks := make([]int, 0, len(values))
	for _, value := range values {
		chunkIndex, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid uploaded chunk index %q: %w", value, err)
		}
		chunks = append(chunks, chunkIndex)
	}
	sort.Ints(chunks)
	return chunks, nil
}

func (h *ChunkUploadHandler) hasUploadedChunk(ctx context.Context, uploadID string, chunkIndex int) (bool, error) {
	chunks, err := h.uploadedChunks(ctx, uploadID)
	if err != nil {
		return false, err
	}
	for _, uploaded := range chunks {
		if uploaded == chunkIndex {
			return true, nil
		}
	}
	return false, nil
}

func (h *ChunkUploadHandler) partETags(ctx context.Context, uploadID string) (map[int]string, error) {
	values, err := h.cache.HGetAll(ctx, h.etagKey(uploadID))
	if err != nil {
		return nil, err
	}
	etags := make(map[int]string, len(values))
	for key, etag := range values {
		chunkIndex, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk etag index %q: %w", key, err)
		}
		etags[chunkIndex] = etag
	}
	return etags, nil
}

func (h *ChunkUploadHandler) refreshUploadTTL(ctx context.Context, session *ChunkUploadSession) error {
	if err := h.cache.Expire(ctx, h.sessionKey(session.UploadID), sessionTTL); err != nil {
		return err
	}
	if err := h.cache.Expire(ctx, h.uploadedKey(session.UploadID), sessionTTL); err != nil {
		return err
	}
	if err := h.cache.Expire(ctx, h.etagKey(session.UploadID), sessionTTL); err != nil {
		return err
	}
	return h.cache.Expire(ctx, h.hashKey(session.AccountID, session.FileHash), sessionTTL)
}

func (h *ChunkUploadHandler) recordUploadedPart(
	ctx context.Context,
	session *ChunkUploadSession,
	chunkIndex int,
	etag string,
) error {
	field := strconv.Itoa(chunkIndex)
	// Write the ETag before marking the chunk uploaded, so complete cannot see
	// a finished chunk without the metadata MinIO requires.
	if err := h.cache.HSet(ctx, h.etagKey(session.UploadID), field, etag); err != nil {
		return err
	}
	if err := h.cache.SAdd(ctx, h.uploadedKey(session.UploadID), field); err != nil {
		return err
	}
	return h.refreshUploadTTL(ctx, session)
}

func (h *ChunkUploadHandler) deleteUploadState(ctx context.Context, session *ChunkUploadSession) {
	_ = h.cache.Del(ctx, h.sessionKey(session.UploadID))
	_ = h.cache.Del(ctx, h.uploadedKey(session.UploadID))
	_ = h.cache.Del(ctx, h.etagKey(session.UploadID))
	_ = h.cache.Del(ctx, h.hashKey(session.AccountID, session.FileHash))
}

func (h *ChunkUploadHandler) InitChunkUpload(c *gin.Context) {
	if !h.available(c) {
		return
	}

	var req InitChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const maxSize = 200 << 20
	if req.FileSize > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 200MB limit"})
		return
	}
	if strings.ToLower(filepath.Ext(req.Filename)) != ".mp4" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .mp4 is allowed"})
		return
	}
	expectedChunks := int((req.FileSize + req.ChunkSize - 1) / req.ChunkSize)
	if req.TotalChunks != expectedChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "total_chunks does not match file size"})
		return
	}

	hashKey := h.hashKey(accountID, req.FileHash)
	existingID, err := h.cache.GetBytes(c.Request.Context(), hashKey)
	if err == nil && len(existingID) > 0 {
		session, sessionErr := h.getSession(c, string(existingID))
		if sessionErr == nil &&
			session.AccountID == accountID &&
			session.FileSize == req.FileSize &&
			session.ChunkSize == req.ChunkSize &&
			session.TotalChunks == req.TotalChunks {
			_ = h.cache.SetBytes(c.Request.Context(), hashKey, existingID, sessionTTL)
			_ = h.saveSession(c, session)
			_ = h.refreshUploadTTL(c.Request.Context(), session)
			uploadedChunks, err := h.uploadedChunks(c.Request.Context(), session.UploadID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded chunks"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"upload_id":       session.UploadID,
				"uploaded_chunks": uploadedChunks,
			})
			return
		}
	}

	objectKey, err := storage.NewMediaObjectKey(storage.MediaVideo, accountID, ".mp4", time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate object key"})
		return
	}
	storageUploadID, err := h.storage.NewMultipartUpload(c.Request.Context(), objectKey, "video/mp4")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to initialize object upload"})
		return
	}

	id, err := randHex(16)
	if err != nil {
		_ = h.storage.AbortMultipartUpload(c.Request.Context(), objectKey, storageUploadID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload session"})
		return
	}
	uploadID := id + fmt.Sprintf("%d", time.Now().UnixNano())
	session := &ChunkUploadSession{
		UploadID:        uploadID,
		StorageUploadID: storageUploadID,
		ObjectKey:       objectKey,
		AccountID:       accountID,
		Filename:        req.Filename,
		FileSize:        req.FileSize,
		ChunkSize:       req.ChunkSize,
		TotalChunks:     req.TotalChunks,
		FileHash:        req.FileHash,
	}

	if err := h.saveSession(c, session); err != nil {
		_ = h.storage.AbortMultipartUpload(c.Request.Context(), objectKey, storageUploadID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	if err := h.cache.SetBytes(c.Request.Context(), hashKey, []byte(uploadID), sessionTTL); err != nil {
		_ = h.cache.Del(c.Request.Context(), h.sessionKey(uploadID))
		_ = h.storage.AbortMultipartUpload(c.Request.Context(), objectKey, storageUploadID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_id":       uploadID,
		"uploaded_chunks": []int{},
	})
}

func (h *ChunkUploadHandler) UploadChunk(c *gin.Context) {
	if !h.available(c) {
		return
	}

	var req UploadChunkRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if req.ChunkIndex < 0 || req.ChunkIndex >= session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk_index"})
		return
	}
	uploaded, err := h.hasUploadedChunk(c.Request.Context(), req.UploadID, req.ChunkIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded chunks"})
		return
	}
	if uploaded {
		_ = h.refreshUploadTTL(c.Request.Context(), session)
		c.JSON(http.StatusOK, gin.H{"chunk_index": req.ChunkIndex})
		return
	}

	formFile, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	expectedSize := session.ChunkSize
	if req.ChunkIndex == session.TotalChunks-1 {
		expectedSize = session.FileSize - int64(req.ChunkIndex)*session.ChunkSize
	}
	if formFile.Size != expectedSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk size"})
		return
	}

	chunkFile, err := formFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}
	defer chunkFile.Close()

	data, err := io.ReadAll(io.LimitReader(chunkFile, expectedSize+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}
	if int64(len(data)) != expectedSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk size"})
		return
	}
	actualHash := fmt.Sprintf("%x", md5.Sum(data))
	if actualHash != req.ChunkHash {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "chunk hash mismatch",
			"expected": req.ChunkHash,
			"actual":   actualHash,
		})
		return
	}

	part, err := h.storage.PutObjectPart(
		c.Request.Context(),
		session.ObjectKey,
		session.StorageUploadID,
		req.ChunkIndex+1,
		bytes.NewReader(data),
		int64(len(data)),
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to store chunk"})
		return
	}

	if err := h.recordUploadedPart(c.Request.Context(), session, req.ChunkIndex, part.ETag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunk_index": req.ChunkIndex})
}

func (h *ChunkUploadHandler) ChunkStatus(c *gin.Context) {
	if !h.available(c) {
		return
	}

	var req ChunkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	uploadedChunks, err := h.uploadedChunks(c.Request.Context(), session.UploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded chunks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_id":       session.UploadID,
		"uploaded_chunks": uploadedChunks,
		"total_chunks":    session.TotalChunks,
	})
}

func (h *ChunkUploadHandler) CompleteChunkUpload(c *gin.Context) {
	if !h.available(c) {
		return
	}

	var req CompleteChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	completed, err := h.cache.SCard(c.Request.Context(), h.uploadedKey(session.UploadID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded chunks"})
		return
	}
	if completed != int64(session.TotalChunks) {
		missing := session.TotalChunks - int(completed)
		if missing < 0 {
			missing = 0
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "not all chunks uploaded",
			"missing":   missing,
			"completed": int(completed),
			"total":     session.TotalChunks,
		})
		return
	}

	etags, err := h.partETags(c.Request.Context(), session.UploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded chunks"})
		return
	}
	parts := make([]storage.CompletedPart, 0, session.TotalChunks)
	for chunkIndex := 0; chunkIndex < session.TotalChunks; chunkIndex++ {
		etag := etags[chunkIndex]
		if etag == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "upload session is missing part metadata"})
			return
		}
		parts = append(parts, storage.CompletedPart{PartNumber: chunkIndex + 1, ETag: etag})
	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	if _, err := h.storage.CompleteMultipartUpload(
		c.Request.Context(),
		session.ObjectKey,
		session.StorageUploadID,
		parts,
		"video/mp4",
	); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to complete object upload"})
		return
	}

	playURL, err := h.storage.PresignedGetURL(c.Request.Context(), session.ObjectKey, h.signedURLTTL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create video URL"})
		return
	}

	h.deleteUploadState(c.Request.Context(), session)

	c.JSON(http.StatusOK, gin.H{
		"object_key": session.ObjectKey,
		"url":        playURL,
		"play_url":   playURL,
	})
}

func (h *ChunkUploadHandler) AbortChunkUpload(c *gin.Context) {
	if !h.available(c) {
		return
	}

	var req AbortChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if err := h.storage.AbortMultipartUpload(c.Request.Context(), session.ObjectKey, session.StorageUploadID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to abort object upload"})
		return
	}

	h.deleteUploadState(c.Request.Context(), session)
	c.JSON(http.StatusOK, gin.H{"message": "upload aborted"})
}
