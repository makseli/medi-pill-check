package models

import (
	"time"

	"gorm.io/gorm"
)

type Medication struct {
	ID           uint           `gorm:"primaryKey;column:id" json:"id"`
	UserID       uint           `gorm:"not null;column:user_id" json:"user_id"`
	Type         int            `gorm:"not null;column:type" json:"type"` // 1=medicine, 2=injectable, 3=other
	Name         string         `gorm:"not null;column:name" json:"name"`
	Dose         string         `gorm:"not null;column:dose" json:"dose"`
	ScheduleType string         `gorm:"not null;column:schedule_type" json:"schedule_type"` // hourly, daily, weekly, monthly
	Description  string         `gorm:"column:description" json:"description"`
	CreatedAt    time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`
}
