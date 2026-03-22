package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"-"`
	Email        string    `gorm:"size:120;uniqueIndex;not null" json:"-"`
	Username     string    `gorm:"size:100;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Status       string    `gorm:"size:20;default:active;not null" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	APIKeys []APIKey `gorm:"foreignKey:UserID" json:"api_keys,omitempty"` //omitempty:如果为空不返回这个字段
}
