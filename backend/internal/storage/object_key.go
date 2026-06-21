package storage

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MediaKind string

const (
	MediaVideo  MediaKind = "videos"
	MediaCover  MediaKind = "covers"
	MediaAvatar MediaKind = "avatars"
)

func NewMediaObjectKey(kind MediaKind, accountID uint, extension string, now time.Time) (string, error) {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension == "" || !strings.HasPrefix(extension, ".") || strings.ContainsAny(extension, `/\`) {
		return "", fmt.Errorf("invalid media extension %q", extension)
	}

	account := fmt.Sprintf("%d", accountID)
	switch kind {
	case MediaVideo, MediaCover:
		return path.Join(string(kind), account, now.Format("20060102"), uuid.NewString()+extension), nil
	case MediaAvatar:
		return path.Join(string(kind), account, "current"+extension), nil
	default:
		return "", fmt.Errorf("unsupported media kind %q", kind)
	}
}
