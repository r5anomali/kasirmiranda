package handlers

import (
	"net/http"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TransactionHandler struct {
	DB *gorm.DB
}

type transactionResponse struct {
	ID            uint      `json:"id"`
	InvoiceNumber string    `json:"invoiceNumber"`
	ProductName   string    `json:"productName"`
	Quantity      int       `json:"quantity"`
	UnitPrice     float64   `json:"unitPrice"`
	Total         float64   `json:"total"`
	DateLabel     string    `json:"dateLabel"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (h *TransactionHandler) Income(c *gin.Context) {
	period := c.DefaultQuery("period", "today")
	start, end, err := periodRange(period, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := h.DB.Model(&models.Transaction{}).Where("type = ? AND created_at >= ? AND created_at < ?", "income", start, end)
	var totalIncome float64
	if err := h.DB.Table("transactions").Where("type = ? AND created_at >= ? AND created_at < ?", "income", start, end).
		Select("COALESCE(SUM(total), 0)").Scan(&totalIncome).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not calculate income"})
		return
	}

	var transactionCount int64
	if err := query.Count(&transactionCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not count transactions"})
		return
	}

	var rows []models.Transaction
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve transactions"})
		return
	}

	transactions := make([]transactionResponse, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, transactionResponse{
			ID: row.ID, InvoiceNumber: row.InvoiceNumber, ProductName: row.ProductName,
			Quantity: row.Quantity, UnitPrice: row.UnitPrice, Total: row.Total,
			DateLabel: dateLabel(row.CreatedAt, time.Now()), CreatedAt: row.CreatedAt,
		})
	}

	average := float64(0)
	if transactionCount > 0 {
		average = totalIncome / float64(transactionCount)
	}
	c.JSON(http.StatusOK, gin.H{"totalIncome": totalIncome, "transactionCount": transactionCount, "averageIncome": average, "transactions": transactions})
}

func periodRange(period string, now time.Time) (time.Time, time.Time, error) {
	location := now.Location()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	switch period {
	case "today":
		return day, day.AddDate(0, 0, 1), nil
	case "week":
		return day.AddDate(0, 0, -6), day.AddDate(0, 0, 1), nil
	case "month":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location), day.AddDate(0, 0, 1), nil
	default:
		return time.Time{}, time.Time{}, &invalidPeriodError{period: period}
	}
}

type invalidPeriodError struct{ period string }

func (e *invalidPeriodError) Error() string { return "period must be today, week, or month" }

func dateLabel(createdAt, now time.Time) string {
	if createdAt.Year() == now.Year() && createdAt.YearDay() == now.YearDay() {
		return "Hari ini, " + createdAt.Format("15:04")
	}
	return createdAt.Format("02 Jan 2006, 15:04")
}
