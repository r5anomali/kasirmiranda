package models

import "gorm.io/gorm"

type Product struct {
	gorm.Model
	Name  string  `json:"name" gorm:"type:varchar(255);not null"`
	Stock int     `json:"stock" gorm:"not null;default:0"`
	Price float64 `json:"price" gorm:"not null;default:0"`
}
