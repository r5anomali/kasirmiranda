package handlers

import (
	"net/http"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
)

type expenseResponse struct {
	ID        uint    `json:"id"`
	Title     string  `json:"title"`
	Amount    float64 `json:"amount"`
	Type      string  `json:"type"` // "debt" or "loan"
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
