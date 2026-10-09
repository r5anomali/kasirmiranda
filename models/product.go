package models

import "gorm.io/gorm"

type Product struct {
	gorm.Model
	SKU           string  `json:"sku" gorm:"type:varchar(32);uniqueIndex;not null"`
	Barcode       *string `json:"barcode" gorm:"type:varchar(64);uniqueIndex"`
	Name          string  `json:"name" gorm:"type:varchar(255);not null"`
	Category      string  `json:"category" gorm:"type:varchar(100);not null"`
	Unit          string  `json:"unit" gorm:"type:varchar(50);not null"`
	PurchasePrice float64 `json:"purchasePrice" gorm:"not null;default:0"`
	SellingPrice  float64 `json:"sellingPrice" gorm:"not null;default:0"`
	Stock         int     `json:"stock" gorm:"not null;default:0"`
	InitialStock  int     `json:"initialStock" gorm:"not null;default:0"`
	StockMinimum  int     `json:"stockMinimum" gorm:"not null;default:0"`
	Active        bool    `json:"active" gorm:"not null;default:true"`
}
