package model

import "time"

type UsageLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	APIKeyID       uint      `gorm:"index;not null" json:"api_key_id"`
	Provider       string    `gorm:"size:50;index;not null" json:"provider"`
	Model          string    `gorm:"size:100;not null" json:"model"`
	RequestTokens  int       `gorm:"default:0;not null" json:"request_tokens"`
	ResponseTokens int       `gorm:"default:0;not null" json:"response_tokens"`
	LatencyMS      int64     `gorm:"default:0;not null" json:"latency_ms"`
	StatusCode     int       `gorm:"default:200;not null" json:"status_code"`
	ErrorMessage   string    `gorm:"type:text" json:"error_message"`
	CreatedAt      time.Time `gorm:"index" json:"created_at"`
}

func (UsageLog) TableName() string {
	return "usage_logs"
}
