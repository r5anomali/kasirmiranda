package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf/v2"
)

func (h *TransactionHandler) ExportIncome(c *gin.Context) {
	h.exportTransactionsPDF(c, "income")
}

func (h *TransactionHandler) ExportOutcome(c *gin.Context) {
	h.exportTransactionsPDF(c, "outcome")
}

func (h *TransactionHandler) exportTransactionsPDF(c *gin.Context, transactionType string) {
	period := c.DefaultQuery("period", "today")
	start, end, err := periodRange(period, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := h.DB.Model(&models.Transaction{}).
		Where("type = ? AND created_at >= ? AND created_at < ?", transactionType, start, end)
	if transactionType == "outcome" {
		subType := c.Query("type")
		if subType != "" && subType != "debt" && subType != "loan" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type parameter must be debt or loan"})
			return
		}
		if subType != "" {
			query = query.Where("sub_type = ?", subType)
		}
	}

	var rows []models.Transaction
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve transactions"})
		return
	}

	var total float64
	for _, row := range rows {
		total += row.Total
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(fmt.Sprintf("%s report", transactionType), false)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, fmt.Sprintf("KasirMiranda - %s Report", transactionType))
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 7, fmt.Sprintf("Period: %s | Generated: %s", period, time.Now().Format("02 Jan 2006 15:04")))
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 10)
	widths := []float64{12, 35, 55, 25, 25, 35}
	for i, header := range []string{"ID", "Invoice", "Title/Product", "Type", "Qty", "Amount"} {
		pdf.CellFormat(widths[i], 8, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 9)
	for _, row := range rows {
		title := row.Title
		if title == "" {
			title = row.ProductName
		}
		subType := row.SubType
		if subType == "" {
			subType = "-"
		}
		values := []string{
			fmt.Sprint(row.ID), row.InvoiceNumber, title, subType,
			fmt.Sprint(row.Quantity), formatIDR(row.Total),
		}
		for i, value := range values {
			pdf.CellFormat(widths[i], 8, value, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(0, 8, fmt.Sprintf("Total: %s (%d transactions)", formatIDR(total), len(rows)))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate PDF"})
		return
	}

	filename := fmt.Sprintf("%s-%s.pdf", transactionType, period)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}

func formatIDR(amount float64) string {
	value := int64(amount + 0.5)
	if value == 0 {
		return "Rp0"
	}

	negative := value < 0
	if negative {
		value = -value
	}

	digits := fmt.Sprint(value)
	grouped := ""
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped += "."
		}
		grouped += string(digit)
	}

	if negative {
		return "Rp.-" + grouped
	}
	return "Rp." + grouped
}
