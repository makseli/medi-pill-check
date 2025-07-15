package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID                uint           `gorm:"primaryKey;column:id" json:"id"`
	Username          string         `gorm:"unique;not null;column:username" json:"username"`
	Email             string         `gorm:"unique;not null;column:email" json:"email"`
	Password          string         `gorm:"not null;column:password" json:"-"`
	MainUser          *uint          `gorm:"column:main_user;default:null" json:"main_user"` // If this user is a dependent, stores the main user's ID
	PasswordChangedAt *time.Time     `gorm:"column:password_changed_at" json:"password_changed_at,omitempty"`
	CreatedAt         time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`
}
