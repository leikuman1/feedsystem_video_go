package video

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"
	"feedsystem_video_go/internal/storage"

	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	service        *VideoService
	accountService *account.AccountService
	storage        storage.ObjectStorage
	signedURLTTL   time.Duration
}

func NewVideoHandler(
	service *VideoService,
	accountService *account.AccountService,
	objectStore storage.ObjectStorage,
	signedURLTTL time.Duration,
) *VideoHandler {
	return &VideoHandler{
		service:        service,
		accountService: accountService,
		storage:        objectStore,
		signedURLTTL:   signedURLTTL,
	}
}

func (vh *VideoHandler) PublishVideo(c *gin.Context) {
	var req PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	username, err := jwt.GetUsername(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if !storage.IsOwnedMediaObjectKey(req.PlayObjectKey, storage.MediaVideo, authorId) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid play_object_key"})
		return
	}
	if !storage.IsOwnedMediaObjectKey(req.CoverObjectKey, storage.MediaCover, authorId) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cover_object_key"})
		return
	}
	video := &Video{
		AuthorID:       authorId,
		Username:       username,
		Title:          req.Title,
		Description:    req.Description,
		PlayObjectKey:  req.PlayObjectKey,
		CoverObjectKey: req.CoverObjectKey,
		CreateTime:     time.Now(),
	}
	if err := vh.service.Publish(c.Request.Context(), video); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	response, err := vh.presentVideo(c.Request.Context(), video)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create media URL"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (vh *VideoHandler) UploadVideo(c *gin.Context) {
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	const maxSize = 200 << 20
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file size"})
		return
	}

	ext := strings.ToLower(filepath.Ext(f.Filename))
	if ext != ".mp4" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .mp4 is allowed"})
		return
	}

	key, err := storage.NewMediaObjectKey(storage.MediaVideo, authorId, ext, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate object key"})
		return
	}

	file, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read upload"})
		return
	}
	defer file.Close()

	contentType := f.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
	}
	if _, err := vh.storage.PutObject(c.Request.Context(), key, file, f.Size, contentType); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to store video"})
		return
	}

	signedURL, err := vh.storage.PresignedGetURL(c.Request.Context(), key, vh.signedURLTTL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create video URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object_key": key,
		"url":        signedURL,
		"play_url":   signedURL,
	})
}

func (vh *VideoHandler) UploadCover(c *gin.Context) {
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	const maxSize = 10 << 20
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file size"})
		return
	}

	ext := strings.ToLower(filepath.Ext(f.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .jpg/.jpeg/.png/.webp is allowed"})
		return
	}

	key, err := storage.NewMediaObjectKey(storage.MediaCover, authorId, ext, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate object key"})
		return
	}

	file, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read upload"})
		return
	}
	defer file.Close()

	contentType := f.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
	}
	if _, err := vh.storage.PutObject(c.Request.Context(), key, file, f.Size, contentType); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to store cover"})
		return
	}

	signedURL, err := vh.storage.PresignedGetURL(c.Request.Context(), key, vh.signedURLTTL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create cover URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object_key": key,
		"url":        signedURL,
		"cover_url":  signedURL,
	})
}

func randHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func buildAbsoluteURL(c *gin.Context, p string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if xf := c.GetHeader("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, p)
}

func (vh *VideoHandler) DeleteVideo(c *gin.Context) {
	var req DeleteVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := vh.service.Delete(c.Request.Context(), req.ID, authorId); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "video deleted"})
}

func (vh *VideoHandler) ListByAuthorID(c *gin.Context) {
	var req ListByAuthorIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	videos, err := vh.service.ListByAuthorID(c.Request.Context(), req.AuthorID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if videos == nil {
		videos = []Video{}
	}
	responses := make([]VideoResponse, 0, len(videos))
	for i := range videos {
		response, err := vh.presentVideo(c.Request.Context(), &videos[i])
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create media URL"})
			return
		}
		responses = append(responses, response)
	}
	c.JSON(http.StatusOK, responses)
}

func (vh *VideoHandler) GetDetail(c *gin.Context) {
	var req GetDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	video, err := vh.service.GetDetail(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	response, err := vh.presentVideo(c.Request.Context(), video)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create media URL"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (vh *VideoHandler) UpdateLikesCount(c *gin.Context) {
	var req UpdateLikesCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := vh.service.UpdateLikesCount(c.Request.Context(), req.ID, req.LikesCount); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "likes count updated"})
}

func (vh *VideoHandler) presentVideo(ctx context.Context, video *Video) (VideoResponse, error) {
	resolver := storage.NewURLResolver(vh.storage, vh.signedURLTTL)
	playURL, err := resolver.Resolve(ctx, video.PlayObjectKey, video.PlayURL)
	if err != nil {
		return VideoResponse{}, err
	}
	coverURL, err := resolver.Resolve(ctx, video.CoverObjectKey, video.CoverURL)
	if err != nil {
		return VideoResponse{}, err
	}
	return VideoResponse{
		ID:          video.ID,
		AuthorID:    video.AuthorID,
		Username:    video.Username,
		Title:       video.Title,
		Description: video.Description,
		PlayURL:     playURL,
		CoverURL:    coverURL,
		CreateTime:  video.CreateTime,
		LikesCount:  video.LikesCount,
		Popularity:  video.Popularity,
	}, nil
}
