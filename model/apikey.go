package model

import "time"

type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	Name       string     `gorm:"size:100;not null" json:"name"`
	LegacyHash string     `gorm:"column:key;size:64;uniqueIndex;not null" json:"-"`
	KeyHash    string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	KeyPreview string     `gorm:"size:32;not null" json:"key_preview"`
	Status     string     `gorm:"size:20;default:active;not null" json:"status"`
	RateLimit  int        `gorm:"default:60;not null" json:"rate_limit"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (APIKey) TableName() string {
	return "apikeys"
}
