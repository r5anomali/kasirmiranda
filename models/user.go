package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username     string `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"type:varchar(255);column:password_hash;not null"`
	Role         string `json:"role" gorm:"not null;default:user"`
}
