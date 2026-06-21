package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"feedsystem_video_go/internal/auth"
	rediscache "feedsystem_video_go/internal/middleware/redis"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

func TestJWTAuthAcceptsQueryTokenForSSE(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-test-secret-with-sufficient-length")
	gin.SetMode(gin.TestMode)

	mr := miniredis.RunT(t)
	cache := rediscache.NewClient(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), "")
	t.Cleanup(func() { _ = cache.Close() })

	token, err := auth.GenerateToken(7, "demo")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if err := cache.SetBytes(
		t.Context(),
		cache.Key("account:%d", 7),
		[]byte(token),
		time.Hour,
	); err != nil {
		t.Fatalf("cache token: %v", err)
	}

	router := gin.New()
	router.GET("/stream", JWTAuth(nil, cache), func(c *gin.Context) {
		accountID, _ := GetAccountID(c)
		c.JSON(http.StatusOK, gin.H{"account_id": accountID})
	})

	request := httptest.NewRequest(http.MethodGet, "/stream?token="+token, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestJWTAuthRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/private", JWTAuth(nil, nil), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}
