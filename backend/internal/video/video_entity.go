package video

import "time"

type Video struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	AuthorID       uint      `gorm:"index;not null" json:"author_id"`
	Username       string    `gorm:"type:varchar(255);not null" json:"username"`
	Title          string    `gorm:"type:varchar(255);not null" json:"title"`
	Description    string    `gorm:"type:varchar(255);" json:"description,omitempty"`
	PlayObjectKey  string    `gorm:"type:varchar(512);index" json:"-"`
	CoverObjectKey string    `gorm:"type:varchar(512);index" json:"-"`
	PlayURL        string    `gorm:"type:varchar(512);not null" json:"-"`
	CoverURL       string    `gorm:"type:varchar(512);not null" json:"-"`
	CreateTime     time.Time `gorm:"autoCreateTime;index:idx_videos_create_time,sort:desc;index:idx_videos_popularity_time_id,priority:2,sort:desc" json:"create_time"`
	LikesCount     int64     `gorm:"column:likes_count;not null;default:0;index:idx_videos_likes_count_id,priority:1,sort:desc" json:"likes_count"`
	Popularity     int64     `gorm:"column:popularity;not null;default:0;index:idx_videos_popularity_time_id,priority:1,sort:desc" json:"popularity"`
}

type PublishVideoRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	PlayObjectKey  string `json:"play_object_key"`
	CoverObjectKey string `json:"cover_object_key"`
}

type VideoResponse struct {
	ID          uint      `json:"id"`
	AuthorID    uint      `json:"author_id"`
	Username    string    `json:"username"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	PlayURL     string    `json:"play_url"`
	CoverURL    string    `json:"cover_url"`
	CreateTime  time.Time `json:"create_time"`
	LikesCount  int64     `json:"likes_count"`
	Popularity  int64     `json:"popularity"`
}

type DeleteVideoRequest struct {
	ID uint `json:"id"`
}

type ListByAuthorIDRequest struct {
	AuthorID uint `json:"author_id"`
}

type GetDetailRequest struct {
	ID uint `json:"id"`
}

type UpdateLikesCountRequest struct {
	ID         uint  `json:"id"`
	LikesCount int64 `json:"likes_count"`
}

type OutboxMsg struct {
	ID         uint      `gorm:"primaryKey"`
	VideoID    uint      `gorm:"index"`
	EventType  string    `gorm:"type:varchar(50)"`
	CreateTime time.Time `gorm:"autoCreateTime"`
	Status     string    `gorm:"type:varchar(50);index"`
}
