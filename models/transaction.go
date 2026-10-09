package models

import "gorm.io/gorm"

type Transaction struct {
	gorm.Model
	Type          string  `json:"type" gorm:"type:varchar(32);index;not null"`                   // "income" or "outcome"
	SubType       string  `json:"subType" gorm:"type:varchar(32);index;default:''"`              // "debt" (hutang) or "loan" (pinjam)
	Status        string  `json:"status" gorm:"type:varchar(32);index;not null;default:pending"` // "pending" or "paid"
	Title         string  `json:"title" gorm:"type:varchar(255);default:''"`                     // e.g. "Supplier Beras"
	Note          string  `json:"note" gorm:"type:varchar(255);default:''"`                      // e.g. "Pembayaran hutang bulan ini"
	InvoiceNumber string  `json:"invoiceNumber" gorm:"type:varchar(64);default:''"`
	ProductName   string  `json:"productName" gorm:"type:varchar(255);default:''"`
	Quantity      int     `json:"quantity" gorm:"not null;default:1"`
	UnitPrice     float64 `json:"unitPrice" gorm:"not null;default:0"`
	Total         float64 `json:"total" gorm:"not null;default:0"`
}
