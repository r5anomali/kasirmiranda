package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type expenseResponse struct {
	ID        uint    `json:"id"`
	Title     string  `json:"title"`
	Amount    float64 `json:"amount"`
	Type      string  `json:"type"` // "debt" or "loan"
	Status    string  `json:"status"`
	Note      string  `json:"note"`
	DateLabel string  `json:"dateLabel"`
}

func (h *TransactionHandler) Outcome(c *gin.Context) {
	period := c.DefaultQuery("period", "today")
	start, end, err := periodRange(period, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subTypeFilter := c.Query("type")
	if subTypeFilter != "" && subTypeFilter != "debt" && subTypeFilter != "loan" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type parameter must be debt or loan"})
		return
	}

	whereClause := "type = ? AND created_at >= ? AND created_at < ?"
	args := []interface{}{"outcome", start, end}

	if subTypeFilter != "" {
		whereClause += " AND sub_type = ?"
		args = append(args, subTypeFilter)
	}

	query := h.DB.Model(&models.Transaction{}).Where(whereClause, args...)

	var totalExpense float64
	if err := h.DB.Table("transactions").
		Where(whereClause, args...).
		Select("COALESCE(SUM(total), 0)").Scan(&totalExpense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate expense"})
		return
	}

	var transactionCount int64
	if err := query.Count(&transactionCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not count expense transactions"})
		return
	}

	var rows []models.Transaction
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve expense transactions"})
		return
	}

	expenses := make([]expenseResponse, 0, len(rows))
	for _, row := range rows {
		title := row.Title
		if title == "" {
			title = row.ProductName
		}
		subType := row.SubType
		if subType == "" {
			subType = "debt"
		}
		expenses = append(expenses, expenseResponse{
			ID:        row.ID,
			Title:     title,
			Amount:    row.Total,
			Type:      subType,
			Status:    row.Status,
			Note:      row.Note,
			DateLabel: dateLabel(row.CreatedAt, time.Now()),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"totalExpense":     totalExpense,
		"transactionCount": transactionCount,
		"expenses":         expenses,
	})
}

var errNotOutcome = errors.New("transaction is not an outcome")
var errAlreadyPaid = errors.New("transaction already paid")

type loanRequest struct {
	Title  string  `json:"title" binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Note   string  `json:"note"`
}

func (h *TransactionHandler) CreateLoan(c *gin.Context) {
	var input loanRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoiceNumber := fmt.Sprintf("LOAN-%s", time.Now().Format("20060102150405"))
	loan := models.Transaction{
		Type: "outcome", Status: "pending", SubType: "loan",
		Title: input.Title, Note: input.Note, InvoiceNumber: invoiceNumber,
		Quantity: 1, UnitPrice: input.Amount, Total: input.Amount,
	}

	if err := h.DB.Create(&loan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create loan"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id": loan.ID, "title": loan.Title, "amount": loan.Total,
		"type": "loan", "status": loan.Status, "note": loan.Note,
		"invoiceNumber": loan.InvoiceNumber,
	})
}

func (h *TransactionHandler) PayDebt(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	var outcome models.Transaction
	invoiceNumber := fmt.Sprintf("INV-%s", time.Now().Format("20060102150405.000000"))
	paymentStatus := ""

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&outcome, id).Error; err != nil {
			return err
		}
		if outcome.Type != "outcome" {
			return errNotOutcome
		}
		if outcome.Status == "paid" {
			return errAlreadyPaid
		}

		paymentStatus = outcome.Status
		outcome.Status = "paid"
		if err := tx.Save(&outcome).Error; err != nil {
			return err
		}

		income := models.Transaction{
			Type: "income", Status: "paid", SubType: "",
			Title: outcome.Title, Note: outcome.Note, InvoiceNumber: invoiceNumber,
			ProductName: outcome.ProductName, Quantity: outcome.Quantity,
			UnitPrice: outcome.UnitPrice, Total: outcome.Total,
		}
		return tx.Create(&income).Error
	})

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if errors.Is(err, errNotOutcome) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "transaction is not an outcome"})
		return
	}
	if errors.Is(err, errAlreadyPaid) {
		c.JSON(http.StatusConflict, gin.H{"error": "transaction already paid"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment recorded", "invoiceNumber": invoiceNumber, "status": paymentStatus})
}
