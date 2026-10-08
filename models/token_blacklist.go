package models

import (
	"time"

	"gorm.io/gorm"
)

type TokenBlacklist struct {
	gorm.Model
	Signature string    `json:"-" gorm:"type:varchar(512);uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expiresAt" gorm:"index;not null"`
}
