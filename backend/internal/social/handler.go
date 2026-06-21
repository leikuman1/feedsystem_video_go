package social

import (
	"context"
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"
	"feedsystem_video_go/internal/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SocialHandler struct {
	service       *SocialService
	mediaResolver *storage.URLResolver
}

func NewSocialHandler(service *SocialService, mediaResolver *storage.URLResolver) *SocialHandler {
	return &SocialHandler{service: service, mediaResolver: mediaResolver}
}

func (h *SocialHandler) Follow(c *gin.Context) {
	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VloggerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	FollowerID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	social := &Social{
		FollowerID: FollowerID,
		VloggerID:  req.VloggerID,
	}
	if err := h.service.Follow(c.Request.Context(), social); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "followed"})
}

func (h *SocialHandler) Unfollow(c *gin.Context) {
	var req UnfollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VloggerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	FollowerID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	social := &Social{
		FollowerID: FollowerID,
		VloggerID:  req.VloggerID,
	}
	if err := h.service.Unfollow(c.Request.Context(), social); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

func (h *SocialHandler) GetAllFollowers(c *gin.Context) {
	var req GetAllFollowersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	vloggerID := req.VloggerID
	if vloggerID == 0 {
		accountID, err := jwt.GetAccountID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		vloggerID = accountID
	}

	followers, err := h.service.GetAllFollowers(c.Request.Context(), vloggerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	presentedFollowers, err := h.presentAccounts(c.Request.Context(), followers)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create avatar URL"})
		return
	}
	followerCount, _ := h.service.CountFollowers(c.Request.Context(), vloggerID)
	c.JSON(http.StatusOK, GetAllFollowersResponse{Followers: presentedFollowers, FollowerCount: followerCount})
}

func (h *SocialHandler) GetAllVloggers(c *gin.Context) {
	var req GetAllVloggersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	followerID := req.FollowerID
	if followerID == 0 {
		accountID, err := jwt.GetAccountID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		followerID = accountID
	}

	vloggers, err := h.service.GetAllVloggers(c.Request.Context(), followerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	presentedVloggers, err := h.presentAccounts(c.Request.Context(), vloggers)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create avatar URL"})
		return
	}
	vloggerCount, _ := h.service.CountVloggers(c.Request.Context(), followerID)
	c.JSON(http.StatusOK, GetAllVloggersResponse{Vloggers: presentedVloggers, VloggerCount: vloggerCount})
}

func (h *SocialHandler) GetCounts(c *gin.Context) {
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	followerCount, _ := h.service.CountFollowers(c.Request.Context(), accountID)
	vloggerCount, _ := h.service.CountVloggers(c.Request.Context(), accountID)
	c.JSON(http.StatusOK, SocialCounts{FollowerCount: followerCount, VloggerCount: vloggerCount})
}

func (h *SocialHandler) presentAccounts(
	ctx context.Context,
	accounts []*account.Account,
) ([]account.FindByIDResponse, error) {
	result := make([]account.FindByIDResponse, 0, len(accounts))
	for _, item := range accounts {
		avatarURL, err := h.mediaResolver.Resolve(ctx, item.AvatarObjectKey, item.AvatarURL)
		if err != nil {
			return nil, err
		}
		result = append(result, account.FindByIDResponse{
			ID:        item.ID,
			Username:  item.Username,
			AvatarURL: avatarURL,
			Bio:       item.Bio,
		})
	}
	return result, nil
}
