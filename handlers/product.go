package handlers

import (
	"net/http"
	"strconv"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductHandler struct {
	DB *gorm.DB
}

type productResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Stock     int       `json:"stock"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *ProductHandler) List(c *gin.Context) {
	page, err := queryInt(c, "page", 1, 1, 1_000_000)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit, err := queryInt(c, "limit", 20, 1, 100)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lowStockOnly := false
	if raw := c.Query("lowStockOnly"); raw != "" {
		lowStockOnly, err = strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lowStockOnly must be true or false"})
			return
		}
	}

	query := h.DB.Model(&models.Product{})
	if lowStockOnly {
		query = query.Where("stock <= ?", 5)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not count products"})
		return
	}

	var products []models.Product
	offset := (page - 1) * limit
	if err := query.Order("id ASC").Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve products"})
		return
	}

	items := make([]productResponse, 0, len(products))
	for _, product := range products {
		items = append(items, productResponse{
			ID: product.ID, Name: product.Name, Stock: product.Stock, Price: product.Price,
			CreatedAt: product.CreatedAt, UpdatedAt: product.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func queryInt(c *gin.Context, key string, defaultValue, min, max int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return 0, &queryParameterError{key: key, min: min, max: max}
	}
	return value, nil
}

type queryParameterError struct {
	key string
	min int
	max int
}

func (e *queryParameterError) Error() string {
	return e.key + " must be an integer between " + strconv.Itoa(e.min) + " and " + strconv.Itoa(e.max)
}
