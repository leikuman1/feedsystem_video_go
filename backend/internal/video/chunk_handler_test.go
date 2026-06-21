package video

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/storage"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// ── helpers ──

const testAccountID uint = 1

type fakeMultipartUpload struct {
	key   string
	parts map[int][]byte
}

type fakeObjectStorage struct {
	mu             sync.Mutex
	nextUploadID   int
	uploads        map[string]*fakeMultipartUpload
	objects        map[string][]byte
	failPartUpload bool
}

func newFakeObjectStorage() *fakeObjectStorage {
	return &fakeObjectStorage{
		uploads: make(map[string]*fakeMultipartUpload),
		objects: make(map[string][]byte),
	}
}

func (s *fakeObjectStorage) EnsureReady(context.Context) error {
	return nil
}

func (s *fakeObjectStorage) PutObject(
	_ context.Context,
	key string,
	reader io.Reader,
	_ int64,
	_ string,
) (storage.ObjectInfo, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = data
	return storage.ObjectInfo{Key: key, ETag: computeMD5(data), Size: int64(len(data))}, nil
}

func (s *fakeObjectStorage) NewMultipartUpload(_ context.Context, key, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextUploadID++
	uploadID := fmt.Sprintf("storage-upload-%d", s.nextUploadID)
	s.uploads[uploadID] = &fakeMultipartUpload{key: key, parts: make(map[int][]byte)}
	return uploadID, nil
}

func (s *fakeObjectStorage) PutObjectPart(
	_ context.Context,
	_ string,
	uploadID string,
	partNumber int,
	reader io.Reader,
	_ int64,
) (storage.CompletedPart, error) {
	if s.failPartUpload {
		return storage.CompletedPart{}, errors.New("part upload failed")
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return storage.CompletedPart{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	upload := s.uploads[uploadID]
	if upload == nil {
		return storage.CompletedPart{}, errors.New("upload not found")
	}
	upload.parts[partNumber] = data
	return storage.CompletedPart{PartNumber: partNumber, ETag: computeMD5(data)}, nil
}

func (s *fakeObjectStorage) CompleteMultipartUpload(
	_ context.Context,
	key, uploadID string,
	parts []storage.CompletedPart,
	_ string,
) (storage.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	upload := s.uploads[uploadID]
	if upload == nil {
		return storage.ObjectInfo{}, errors.New("upload not found")
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].PartNumber < parts[j].PartNumber })
	var merged bytes.Buffer
	for _, part := range parts {
		merged.Write(upload.parts[part.PartNumber])
	}
	data := append([]byte(nil), merged.Bytes()...)
	s.objects[key] = data
	delete(s.uploads, uploadID)
	return storage.ObjectInfo{Key: key, ETag: computeMD5(data), Size: int64(len(data))}, nil
}

func (s *fakeObjectStorage) AbortMultipartUpload(_ context.Context, _ string, uploadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.uploads, uploadID)
	return nil
}

func (s *fakeObjectStorage) RemoveObject(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

func (s *fakeObjectStorage) PresignedGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://media.test/" + key, nil
}

func setupTestEnv(t *testing.T) (*ChunkUploadHandler, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	client := rediscache.NewClient(
		goredis.NewClient(&goredis.Options{Addr: mr.Addr()}),
		"",
	)
	handler := NewChunkUploadHandler(client, newFakeObjectStorage(), 2*time.Hour)

	cleanup := func() {
		client.Close()
		mr.Close()
	}
	return handler, cleanup
}

func newJSONContext(t *testing.T, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "localhost"

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Set("accountID", testAccountID)
	return c, rec
}

func newMultipartContext(t *testing.T, path string, fields map[string]string, fileContent []byte) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	part, err := writer.CreateFormFile("file", "chunk.bin")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("write file: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Host = "localhost"

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Set("accountID", testAccountID)
	return c, rec
}

func computeMD5(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])
}

func parseJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("parse json: %v, body: %s", err, rec.Body.String())
	}
	return m
}

func makeTestChunks(t *testing.T, totalChunks, chunkSize int) ([][]byte, []string, string) {
	t.Helper()
	chunks := make([][]byte, totalChunks)
	chunkHashes := make([]string, totalChunks)
	var full bytes.Buffer
	for i := 0; i < totalChunks; i++ {
		data := make([]byte, chunkSize)
		if _, err := rand.Read(data); err != nil {
			t.Fatalf("rand read: %v", err)
		}
		chunks[i] = data
		chunkHashes[i] = computeMD5(data)
		full.Write(data)
	}
	return chunks, chunkHashes, computeMD5(full.Bytes())
}

func initUpload(t *testing.T, h *ChunkUploadHandler, filename string, fileSize int64, chunkSize int64, totalChunks int, fileHash string) string {
	t.Helper()
	c, rec := newJSONContext(t, "/video/chunk/init", InitChunkUploadRequest{
		Filename:    filename,
		FileSize:    fileSize,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		FileHash:    fileHash,
	})
	h.InitChunkUpload(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("init: expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
	resp := parseJSON(t, rec)
	return resp["upload_id"].(string)
}

func uploadChunk(t *testing.T, h *ChunkUploadHandler, uploadID string, chunkIndex int, chunkHash string, chunkData []byte) {
	t.Helper()
	c, rec := newMultipartContext(t, "/video/chunk/upload", map[string]string{
		"upload_id":   uploadID,
		"chunk_index": fmt.Sprintf("%d", chunkIndex),
		"chunk_hash":  chunkHash,
	}, chunkData)
	h.UploadChunk(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload chunk %d: expected 200, got %d, body: %s", chunkIndex, rec.Code, rec.Body.String())
	}
}

func completeUpload(t *testing.T, h *ChunkUploadHandler, uploadID string) map[string]interface{} {
	t.Helper()
	c, rec := newJSONContext(t, "/video/chunk/complete", CompleteChunkUploadRequest{UploadID: uploadID})
	h.CompleteChunkUpload(c)
	return parseJSON(t, rec)
}

// readMergedFile extracts the file path from the response URL and reads it.
// URL format: http://localhost/static/videos/<accountID>/<date>/<name>.mp4
func readMergedFile(t *testing.T, h *ChunkUploadHandler, resp map[string]interface{}) []byte {
	t.Helper()
	key := resp["object_key"].(string)
	store := h.storage.(*fakeObjectStorage)
	store.mu.Lock()
	defer store.mu.Unlock()
	data, ok := store.objects[key]
	if !ok {
		t.Fatalf("object %q not found", key)
	}
	return append([]byte(nil), data...)
}

// ── tests ──

func TestFullChunkUploadFlow(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	chunkSize := 1024
	totalChunks := 3
	chunks, chunkHashes, fileHash := makeTestChunks(t, totalChunks, chunkSize)

	uploadID := initUpload(t, h, "test.mp4", int64(totalChunks*chunkSize), int64(chunkSize), totalChunks, fileHash)

	for i := 0; i < totalChunks; i++ {
		uploadChunk(t, h, uploadID, i, chunkHashes[i], chunks[i])
	}

	c, rec := newJSONContext(t, "/video/chunk/complete", CompleteChunkUploadRequest{UploadID: uploadID})
	h.CompleteChunkUpload(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete: expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
	resp := parseJSON(t, rec)
	if resp["url"] == nil || resp["play_url"] == nil {
		t.Fatal("complete response missing url or play_url")
	}

	// verify merged file content matches original chunks
	merged := readMergedFile(t, h, resp)
	var expected bytes.Buffer
	for _, ch := range chunks {
		expected.Write(ch)
	}
	if !bytes.Equal(merged, expected.Bytes()) {
		t.Fatalf("merged file content mismatch: got %d bytes, want %d bytes", len(merged), expected.Len())
	}

}

func TestBreakpointResume(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	chunkSize := 512
	totalChunks := 4
	chunks, chunkHashes, fileHash := makeTestChunks(t, totalChunks, chunkSize)

	uploadID := initUpload(t, h, "resume.mp4", int64(totalChunks*chunkSize), int64(chunkSize), totalChunks, fileHash)

	// upload first 2 chunks
	for i := 0; i < 2; i++ {
		uploadChunk(t, h, uploadID, i, chunkHashes[i], chunks[i])
	}

	// re-init with same file_hash → should resume existing session
	c, rec := newJSONContext(t, "/video/chunk/init", InitChunkUploadRequest{
		Filename:    "resume.mp4",
		FileSize:    int64(totalChunks * chunkSize),
		ChunkSize:   int64(chunkSize),
		TotalChunks: totalChunks,
		FileHash:    fileHash,
	})
	h.InitChunkUpload(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("re-init: expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
	resp := parseJSON(t, rec)

	resumedID := resp["upload_id"].(string)
	if resumedID != uploadID {
		t.Fatalf("resume should return same upload_id: got %s, want %s", resumedID, uploadID)
	}

	chunkList := resp["uploaded_chunks"].([]interface{})
	if len(chunkList) != 2 {
		t.Fatalf("expected 2 uploaded chunks on resume, got %d", len(chunkList))
	}

	// upload remaining chunks and complete
	for i := 2; i < totalChunks; i++ {
		uploadChunk(t, h, uploadID, i, chunkHashes[i], chunks[i])
	}

	resp = completeUpload(t, h, uploadID)
	merged := readMergedFile(t, h, resp)
	var expected bytes.Buffer
	for _, ch := range chunks {
		expected.Write(ch)
	}
	if !bytes.Equal(merged, expected.Bytes()) {
		t.Fatalf("merged file mismatch after resume")
	}
}

func TestIdempotentChunkUpload(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	chunkSize := 256
	totalChunks := 2
	chunks, chunkHashes, fileHash := makeTestChunks(t, totalChunks, chunkSize)

	uploadID := initUpload(t, h, "idempotent.mp4", int64(totalChunks*chunkSize), int64(chunkSize), totalChunks, fileHash)

	// upload chunk 0 twice — both should succeed
	uploadChunk(t, h, uploadID, 0, chunkHashes[0], chunks[0])
	uploadChunk(t, h, uploadID, 0, chunkHashes[0], chunks[0])

	uploadChunk(t, h, uploadID, 1, chunkHashes[1], chunks[1])

	c, rec := newJSONContext(t, "/video/chunk/complete", CompleteChunkUploadRequest{UploadID: uploadID})
	h.CompleteChunkUpload(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete: expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestHashMismatch(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	chunkSize := 256
	totalChunks := 1
	chunks, _, fileHash := makeTestChunks(t, totalChunks, chunkSize)

	uploadID := initUpload(t, h, "hashfail.mp4", int64(chunkSize), int64(chunkSize), totalChunks, fileHash)

	// upload with wrong hash
	c, rec := newMultipartContext(t, "/video/chunk/upload", map[string]string{
		"upload_id":   uploadID,
		"chunk_index": "0",
		"chunk_hash":  "deadbeef000000000000000000000000",
	}, chunks[0])
	h.UploadChunk(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for hash mismatch, got %d", rec.Code)
	}
	resp := parseJSON(t, rec)
	if resp["error"] != "chunk hash mismatch" {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	if resp["expected"] == nil || resp["actual"] == nil {
		t.Fatal("response should contain expected and actual hash")
	}
	actualHash := computeMD5(chunks[0])
	if resp["actual"] != actualHash {
		t.Fatalf("actual hash: got %v, want %s", resp["actual"], actualHash)
	}

	// retry with correct hash should succeed
	uploadChunk(t, h, uploadID, 0, actualHash, chunks[0])
}

func TestIncompleteMerge(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	chunkSize := 128
	totalChunks := 5
	chunks, chunkHashes, fileHash := makeTestChunks(t, totalChunks, chunkSize)

	uploadID := initUpload(t, h, "incomplete.mp4", int64(totalChunks*chunkSize), int64(chunkSize), totalChunks, fileHash)

	// upload only 3 out of 5
	for i := 0; i < 3; i++ {
		uploadChunk(t, h, uploadID, i, chunkHashes[i], chunks[i])
	}

	c, rec := newJSONContext(t, "/video/chunk/complete", CompleteChunkUploadRequest{UploadID: uploadID})
	h.CompleteChunkUpload(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for incomplete, got %d", rec.Code)
	}
	resp := parseJSON(t, rec)
	if resp["error"] != "not all chunks uploaded" {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	if int(resp["missing"].(float64)) != 2 {
		t.Fatalf("expected missing=2, got %v", resp["missing"])
	}
	if int(resp["completed"].(float64)) != 3 {
		t.Fatalf("expected completed=3, got %v", resp["completed"])
	}
	if int(resp["total"].(float64)) != 5 {
		t.Fatalf("expected total=5, got %v", resp["total"])
	}

	// upload remaining and retry
	for i := 3; i < totalChunks; i++ {
		uploadChunk(t, h, uploadID, i, chunkHashes[i], chunks[i])
	}
	c, rec = newJSONContext(t, "/video/chunk/complete", CompleteChunkUploadRequest{UploadID: uploadID})
	h.CompleteChunkUpload(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete after fix: expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestChunkStatus(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	chunkSize := 256
	totalChunks := 3
	chunks, chunkHashes, fileHash := makeTestChunks(t, totalChunks, chunkSize)

	uploadID := initUpload(t, h, "status.mp4", int64(totalChunks*chunkSize), int64(chunkSize), totalChunks, fileHash)

	assertStatus := func(wantCount int) {
		t.Helper()
		c, rec := newJSONContext(t, "/video/chunk/status", ChunkStatusRequest{UploadID: uploadID})
		h.ChunkStatus(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: expected 200, got %d", rec.Code)
		}
		resp := parseJSON(t, rec)
		list, _ := resp["uploaded_chunks"].([]interface{})
		if len(list) != wantCount {
			t.Fatalf("expected %d uploaded chunks, got %d", wantCount, len(list))
		}
		if int(resp["total_chunks"].(float64)) != totalChunks {
			t.Fatalf("expected total_chunks=%d", totalChunks)
		}
	}

	assertStatus(0)

	uploadChunk(t, h, uploadID, 0, chunkHashes[0], chunks[0])
	assertStatus(1)

	uploadChunk(t, h, uploadID, 1, chunkHashes[1], chunks[1])
	uploadChunk(t, h, uploadID, 2, chunkHashes[2], chunks[2])
	assertStatus(3)
}

func TestChunkStorageFailureDoesNotAdvanceSession(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	chunks, chunkHashes, fileHash := makeTestChunks(t, 1, 256)
	uploadID := initUpload(t, h, "failure.mp4", 256, 256, 1, fileHash)
	h.storage.(*fakeObjectStorage).failPartUpload = true

	c, rec := newMultipartContext(t, "/video/chunk/upload", map[string]string{
		"upload_id":   uploadID,
		"chunk_index": "0",
		"chunk_hash":  chunkHashes[0],
	}, chunks[0])
	h.UploadChunk(c)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}

	c, rec = newJSONContext(t, "/video/chunk/status", ChunkStatusRequest{UploadID: uploadID})
	h.ChunkStatus(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	resp := parseJSON(t, rec)
	if len(resp["uploaded_chunks"].([]interface{})) != 0 {
		t.Fatal("failed part must not be marked uploaded")
	}
}

func TestAbortChunkUpload(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _, fileHash := makeTestChunks(t, 1, 128)
	uploadID := initUpload(t, h, "abort.mp4", 128, 128, 1, fileHash)

	c, rec := newJSONContext(t, "/video/chunk/abort", AbortChunkUploadRequest{UploadID: uploadID})
	h.AbortChunkUpload(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("abort: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	c, rec = newJSONContext(t, "/video/chunk/status", ChunkStatusRequest{UploadID: uploadID})
	h.ChunkStatus(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected removed session, got %d", rec.Code)
	}
}
