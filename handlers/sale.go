package handlers

import (
	"fmt"
	"net/http"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type saleItemRequest struct {
	ProductID uint `json:"productId" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,min=1"`
}

type saleRequest struct {
	Items       []saleItemRequest `json:"items" binding:"required,min=1"`
	PaymentType string            `json:"paymentType" binding:"omitempty,oneof=cash debt loan"`
	Title       string            `json:"title"`
	Note        string            `json:"note"`
}

func (h *TransactionHandler) CreateSale(c *gin.Context) {
	var input saleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var responseItems []gin.H
	var grandTotal float64
	invoiceNumber := fmt.Sprintf("INV-%s", time.Now().Format("20060102150405"))

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range input.Items {
			var product models.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&product, item.ProductID).Error; err != nil {
				return fmt.Errorf("product %d not found", item.ProductID)
			}
			if !product.Active {
				return fmt.Errorf("product %s is inactive", product.Name)
			}
			if product.Stock < item.Quantity {
				return fmt.Errorf("insufficient stock for product %s", product.Name)
			}

			lineTotal := product.SellingPrice * float64(item.Quantity)
			product.Stock -= item.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return err
			}

			transactionType := "income"
			status := "paid"
			subType := ""
			if input.PaymentType == "debt" || input.PaymentType == "loan" {
				transactionType = "outcome"
				status = "pending"
				subType = input.PaymentType
			}

			transaction := models.Transaction{
				Type: transactionType, Status: status, SubType: subType,
				Title: input.Title, Note: input.Note, InvoiceNumber: invoiceNumber,
				ProductName: product.Name, Quantity: item.Quantity,
				UnitPrice: product.SellingPrice, Total: lineTotal,
			}
			if err := tx.Create(&transaction).Error; err != nil {
				return err
			}
			grandTotal += lineTotal
			responseItems = append(responseItems, gin.H{
				"productId": product.ID, "productName": product.Name,
				"quantity": item.Quantity, "unitPrice": product.SellingPrice, "total": lineTotal,
			})
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"invoiceNumber": invoiceNumber,
		"items":         responseItems,
		"total":         grandTotal,
	})
}
