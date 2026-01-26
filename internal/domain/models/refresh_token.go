package models

import "time"

type RefreshToken struct {
	Token     string
	UserID    int64
	AppID     int
	ExpiresAt time.Time
	CreatedAt time.Time
}
