package models

import (
	"time"
)

type AuditLog struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`
	UserID    *uint     `gorm:"column:user_id" json:"user_id"`
	Action    string    `gorm:"column:action" json:"action"`
	Detail    string    `gorm:"column:detail" json:"detail"`
	IP        string    `gorm:"column:ip" json:"ip"`
	UserAgent string    `gorm:"column:user_agent" json:"user_agent"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}
