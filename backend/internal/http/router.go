package http

import (
	"context"
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/feed"
	"feedsystem_video_go/internal/message"
	"feedsystem_video_go/internal/middleware/jwt"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	"feedsystem_video_go/internal/middleware/ratelimit"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/social"
	"feedsystem_video_go/internal/storage"
	"feedsystem_video_go/internal/video"
	"feedsystem_video_go/internal/worker"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRouter(
	db *gorm.DB,
	cache *rediscache.Client,
	rmq *rabbitmq.RabbitMQ,
	objectStore storage.ObjectStorage,
	signedURLExpiry time.Duration,
	allowPublicRegistration bool,
) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("SetTrustedProxies failed: %v", err)
	}
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.Static("/static", "./.run/uploads")
	mediaResolver := storage.NewURLResolver(objectStore, signedURLExpiry)
	// rate_limit
	loginLimiter := ratelimit.Limit(cache, "account_login", 10, time.Minute, ratelimit.KeyByIP)
	registerLimiter := ratelimit.Limit(cache, "account_register", 5, time.Hour, ratelimit.KeyByIP)

	likeLimiter := ratelimit.Limit(cache, "like_write", 30, time.Minute, ratelimit.KeyByAccount)
	commentLimiter := ratelimit.Limit(cache, "comment_write", 10, time.Minute, ratelimit.KeyByAccount)
	socialLimiter := ratelimit.Limit(cache, "social_write", 20, time.Minute, ratelimit.KeyByAccount)

	// account
	accountRepository := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepository, cache)
	accountHandler := account.NewAccountHandler(accountService, objectStore, signedURLExpiry)
	accountGroup := r.Group("/account")
	{
		if allowPublicRegistration {
			accountGroup.POST("/register", registerLimiter, accountHandler.CreateAccount)
		} else {
			accountGroup.POST("/register", func(c *gin.Context) {
				c.JSON(403, gin.H{"error": "public registration is disabled"})
			})
		}
		accountGroup.POST("/login", loginLimiter, accountHandler.Login)
		accountGroup.POST("/refresh", accountHandler.Refresh)
	}
	protectedAccountGroup := accountGroup.Group("")
	protectedAccountGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedAccountGroup.POST("/logout", accountHandler.Logout)
		protectedAccountGroup.POST("/rename", accountHandler.Rename)
		protectedAccountGroup.POST("/changePassword", accountHandler.ChangePassword)
		protectedAccountGroup.POST("/findByID", accountHandler.FindByID)
		protectedAccountGroup.POST("/findByUsername", accountHandler.FindByUsername)
		protectedAccountGroup.POST("/uploadAvatar", accountHandler.UploadAvatar)
		protectedAccountGroup.POST("/updateProfile", accountHandler.UpdateProfile)
	}
	// video
	videoRepository := video.NewVideoRepository(db)
	popularityMQ, err := rabbitmq.NewPopularityMQ(rmq)
	if err != nil {
		log.Printf("PopularityMQ init failed (mq disabled): %v", err)
		popularityMQ = nil
	}
	videoService := video.NewVideoService(videoRepository, cache, popularityMQ, objectStore)
	videoHandler := video.NewVideoHandler(videoService, accountService, objectStore, signedURLExpiry)
	chunkHandler := video.NewChunkUploadHandler(cache, objectStore, signedURLExpiry)
	videoGroup := r.Group("/video")
	videoGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		videoGroup.POST("/listByAuthorID", videoHandler.ListByAuthorID)
		videoGroup.POST("/getDetail", videoHandler.GetDetail)
		videoGroup.POST("/uploadVideo", videoHandler.UploadVideo)
		videoGroup.POST("/uploadCover", videoHandler.UploadCover)
		videoGroup.POST("/publish", videoHandler.PublishVideo)
		videoGroup.POST("/chunk/init", chunkHandler.InitChunkUpload)
		videoGroup.POST("/chunk/upload", chunkHandler.UploadChunk)
		videoGroup.POST("/chunk/status", chunkHandler.ChunkStatus)
		videoGroup.POST("/chunk/complete", chunkHandler.CompleteChunkUpload)
		videoGroup.POST("/chunk/abort", chunkHandler.AbortChunkUpload)
	}
	// like
	likeMQ, err := rabbitmq.NewLikeMQ(rmq)
	if err != nil {
		log.Printf("LikeMQ init failed (mq disabled): %v", err)
		likeMQ = nil
	}
	likeRepository := video.NewLikeRepository(db)
	likeService := video.NewLikeService(likeRepository, videoRepository, cache, likeMQ, popularityMQ)
	likeHandler := video.NewLikeHandler(likeService)
	likeGroup := r.Group("/like")
	likeGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		likeGroup.POST("/like", likeLimiter, likeHandler.Like)
		likeGroup.POST("/unlike", likeLimiter, likeHandler.Unlike)
		likeGroup.POST("/isLiked", likeHandler.IsLiked)
		likeGroup.POST("/listMyLikedVideos", likeHandler.ListMyLikedVideos)
	}
	// comment
	commentRepository := video.NewCommentRepository(db)
	commentMQ, err := rabbitmq.NewCommentMQ(rmq)
	if err != nil {
		log.Printf("CommentMQ init failed (mq disabled): %v", err)
		commentMQ = nil
	}
	commentService := video.NewCommentService(commentRepository, videoRepository, cache, commentMQ, popularityMQ)
	commentHandler := video.NewCommentHandler(commentService, accountService)
	commentGroup := r.Group("/comment")
	commentGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		commentGroup.POST("/listAll", commentHandler.GetAllComments)
		commentGroup.POST("/publish", commentLimiter, commentHandler.PublishComment)
		commentGroup.POST("/delete", commentLimiter, commentHandler.DeleteComment)
	}
	// social
	socialMQ, err := rabbitmq.NewSocialMQ(rmq)
	if err != nil {
		log.Printf("SocialMQ init failed (mq disabled): %v", err)
		socialMQ = nil
	}
	socialRepository := social.NewSocialRepository(db)
	socialService := social.NewSocialService(socialRepository, accountRepository, socialMQ, cache)
	socialHandler := social.NewSocialHandler(socialService, mediaResolver)
	socialGroup := r.Group("/social")
	socialGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		socialGroup.POST("/follow", socialLimiter, socialHandler.Follow)
		socialGroup.POST("/unfollow", socialLimiter, socialHandler.Unfollow)
		socialGroup.POST("/getAllFollowers", socialHandler.GetAllFollowers)
		socialGroup.POST("/getAllVloggers", socialHandler.GetAllVloggers)
		socialGroup.POST("/getCounts", socialHandler.GetCounts)
	}

	protectedAccountGroup.POST("/getProfile", func(c *gin.Context) {
		var req account.GetProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if req.AccountID == 0 {
			c.JSON(400, gin.H{"error": "account_id is required"})
			return
		}
		acc, err := accountService.FindByID(c.Request.Context(), req.AccountID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		videoCount, _ := videoRepository.CountByAuthor(c.Request.Context(), req.AccountID)
		totalLikes, _ := videoRepository.TotalLikesByAuthor(c.Request.Context(), req.AccountID)
		followerCount, _ := socialRepository.CountFollowers(c.Request.Context(), req.AccountID)
		vloggerCount, _ := socialRepository.CountVloggers(c.Request.Context(), req.AccountID)

		avatarURL, err := mediaResolver.Resolve(c.Request.Context(), acc.AvatarObjectKey, acc.AvatarURL)
		if err != nil {
			c.JSON(502, gin.H{"error": "failed to create avatar URL"})
			return
		}
		c.JSON(200, account.GetProfileResponse{
			Account:    account.FindByIDResponse{ID: acc.ID, Username: acc.Username, AvatarURL: avatarURL, Bio: acc.Bio},
			VideoCount: videoCount, TotalLikes: totalLikes,
			FollowerCount: followerCount, VloggerCount: vloggerCount,
		})
	})
	// feed
	feedRepository := feed.NewFeedRepository(db)
	feedService := feed.NewFeedService(feedRepository, likeRepository, cache, mediaResolver)
	feedHandler := feed.NewFeedHandler(feedService)
	feedGroup := r.Group("/feed")
	feedGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		feedGroup.POST("/listLatest", feedHandler.ListLatest)
		feedGroup.POST("/listLikesCount", feedHandler.ListLikesCount)
		feedGroup.POST("/listByPopularity", feedHandler.ListByPopularity)
		feedGroup.POST("/listByTag", feedHandler.ListByTag)
		feedGroup.POST("/listByFollowing", feedHandler.ListByFollowing)
	}
	// message
	messageRepo := message.NewRepository(db)
	messageService := message.NewService(messageRepo)
	messageHandler := message.NewHandler(messageService)
	messageGroup := r.Group("/message")
	messageGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		messageGroup.POST("/send", messageHandler.Send)
		messageGroup.POST("/list", messageHandler.List)
	}
	//worker
	timelineMQ, err := rabbitmq.NewTimelineMQ(rmq)
	if err != nil {
		log.Printf("timelineMQ init failed (mq disabled): %v", err)
		timelineMQ = nil
	}
	worker.StartOutboxPoller(db, timelineMQ)
	worker.StartConsumer(timelineMQ, "video.timeline.update.queue", cache, rmq)

	// SSE notification
	if rmq != nil {
		if notifCh, err := rmq.NewChannel(); err == nil {
			if err := rabbitmq.DeclareTopic(notifCh, "like.events", "notification.like", "like.like"); err != nil {
				log.Printf("notification like topic init failed: %v", err)
			}
			if err := rabbitmq.DeclareTopic(notifCh, "comment.events", "notification.comment", "comment.publish"); err != nil {
				log.Printf("notification comment topic init failed: %v", err)
			}
			if err := rabbitmq.DeclareTopic(notifCh, "social.events", "notification.social", "social.follow"); err != nil {
				log.Printf("notification social topic init failed: %v", err)
			}
			notifCh.Close()
		}
	}
	sseHub := worker.NewSSEHub(db)
	notifGroup := r.Group("/notification")
	notifGroup.Use(jwt.JWTAuth(accountRepository, cache))
	sseHub.RegisterRoutes(r, notifGroup)

	go func() {
		if rmq != nil {
			hub := sseHub
			ctx := context.Background()
			// 每个 notification worker 独立 Channel + 自动重连
			for _, q := range []string{"notification.like", "notification.comment", "notification.social"} {
				go func(queue string) {
					for {
						ch, err := rmq.NewChannel()
						if err != nil {
							log.Printf("notification-%s: 创建 Channel 失败: %v, 5秒后重试", queue, err)
							time.Sleep(5 * time.Second)
							continue
						}
						w := worker.NewNotificationWorker(ch, db, queue, hub)
						if err := w.Run(ctx); err != nil {
							log.Printf("notification-%s: %v, 5秒后重连...", queue, err)
						}
						ch.Close()
						time.Sleep(5 * time.Second)
					}
				}(q)
			}
		} else {
			log.Printf("Notification SSE disabled (MQ not available)")
		}
	}()

	return r
}
