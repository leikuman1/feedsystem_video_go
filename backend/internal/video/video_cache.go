package video

import (
	"encoding/json"
	"errors"
	"time"
)

type cachedVideo struct {
	ID             uint      `json:"id"`
	AuthorID       uint      `json:"author_id"`
	Username       string    `json:"username"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	PlayObjectKey  string    `json:"play_object_key,omitempty"`
	CoverObjectKey string    `json:"cover_object_key,omitempty"`
	PlayURL        string    `json:"play_url,omitempty"`
	CoverURL       string    `json:"cover_url,omitempty"`
	CreateTime     time.Time `json:"create_time"`
	LikesCount     int64     `json:"likes_count"`
	Popularity     int64     `json:"popularity"`
}

func MarshalVideoCache(v *Video) ([]byte, error) {
	if v == nil {
		return nil, errors.New("video is nil")
	}
	return json.Marshal(cachedVideo{
		ID:             v.ID,
		AuthorID:       v.AuthorID,
		Username:       v.Username,
		Title:          v.Title,
		Description:    v.Description,
		PlayObjectKey:  v.PlayObjectKey,
		CoverObjectKey: v.CoverObjectKey,
		PlayURL:        v.PlayURL,
		CoverURL:       v.CoverURL,
		CreateTime:     v.CreateTime,
		LikesCount:     v.LikesCount,
		Popularity:     v.Popularity,
	})
}

func UnmarshalVideoCache(b []byte) (*Video, bool) {
	var cached cachedVideo
	if err := json.Unmarshal(b, &cached); err != nil {
		return nil, false
	}
	v := &Video{
		ID:             cached.ID,
		AuthorID:       cached.AuthorID,
		Username:       cached.Username,
		Title:          cached.Title,
		Description:    cached.Description,
		PlayObjectKey:  cached.PlayObjectKey,
		CoverObjectKey: cached.CoverObjectKey,
		PlayURL:        cached.PlayURL,
		CoverURL:       cached.CoverURL,
		CreateTime:     cached.CreateTime,
		LikesCount:     cached.LikesCount,
		Popularity:     cached.Popularity,
	}
	if !HasVideoMediaReferences(v) {
		return nil, false
	}
	return v, true
}

func HasVideoMediaReferences(v *Video) bool {
	if v == nil {
		return false
	}
	return hasMediaReference(v.PlayObjectKey, v.PlayURL) &&
		hasMediaReference(v.CoverObjectKey, v.CoverURL)
}

func hasMediaReference(objectKey, legacyURL string) bool {
	return objectKey != "" || legacyURL != ""
}
