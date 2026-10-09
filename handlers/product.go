package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductHandler struct {
	DB *gorm.DB
}

type productResponse struct {
	ID            uint    `json:"id"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode"`
	Name          string  `json:"name"`
	Category      string  `json:"category"`
	Unit          string  `json:"unit"`
	PurchasePrice float64 `json:"purchasePrice"`
	SellingPrice  float64 `json:"sellingPrice"`
	Stock         int     `json:"stock"`
	InitialStock  int     `json:"initialStock"`
	StockMinimum  int     `json:"stockMinimum"`
	Active        bool    `json:"active"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type productCreateRequest struct {
	SKU           string  `json:"sku" binding:"required"`
	Barcode       string  `json:"barcode"`
	Name          string  `json:"name" binding:"required"`
	Category      string  `json:"category" binding:"required"`
	Unit          string  `json:"unit" binding:"required"`
	PurchasePrice float64 `json:"purchasePrice" binding:"min=0"`
	SellingPrice  float64 `json:"sellingPrice" binding:"min=0"`
	Stock         int     `json:"stock" binding:"min=0"`
	InitialStock  int     `json:"initialStock" binding:"min=0"`
	StockMinimum  int     `json:"stockMinimum" binding:"min=0"`
	Active        *bool   `json:"active"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var input productCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.SKU = strings.TrimSpace(input.SKU)
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	input.Unit = strings.TrimSpace(input.Unit)
	input.Barcode = strings.TrimSpace(input.Barcode)
	if input.SKU == "" || input.Name == "" || input.Category == "" || input.Unit == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sku, name, category, and unit are required"})
		return
	}

	var count int64
	if err := h.DB.Model(&models.Product{}).Where("sku = ?", input.SKU).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not validate product SKU"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "product with this SKU already exists"})
		return
	}
	if input.Barcode != "" {
		if err := h.DB.Model(&models.Product{}).Where("barcode = ?", input.Barcode).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not validate product barcode"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "product with this barcode already exists"})
			return
		}
	}

	now := time.Now()
	createdAt, err := parseOptionalProductDate(input.CreatedAt, now)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "createdAt must use YYYY-MM-DD HH:MM:SS format"})
		return
	}
	updatedAt, err := parseOptionalProductDate(input.UpdatedAt, now)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "updatedAt must use YYYY-MM-DD HH:MM:SS format"})
		return
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	product := models.Product{
		SKU: input.SKU, Name: input.Name, Category: input.Category, Unit: input.Unit,
		PurchasePrice: input.PurchasePrice, SellingPrice: input.SellingPrice,
		Stock: input.Stock, InitialStock: input.InitialStock, StockMinimum: input.StockMinimum,
		Active: active,
	}
	product.CreatedAt = createdAt.Truncate(time.Second)
	product.UpdatedAt = updatedAt.Truncate(time.Second)
	if input.Barcode != "" {
		product.Barcode = &input.Barcode
	}

	if err := h.DB.Create(&product).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "product SKU or barcode already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create product"})
		return
	}

	c.JSON(http.StatusCreated, toProductResponse(product))
}

func parseOptionalProductDate(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	return time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
}

func toProductResponse(product models.Product) productResponse {
	barcode := ""
	if product.Barcode != nil {
		barcode = *product.Barcode
	}
	return productResponse{
		ID: product.ID, SKU: product.SKU, Barcode: barcode, Name: product.Name,
		Category: product.Category, Unit: product.Unit, PurchasePrice: product.PurchasePrice,
		SellingPrice: product.SellingPrice, Stock: product.Stock, InitialStock: product.InitialStock,
		StockMinimum: product.StockMinimum, Active: product.Active,
		CreatedAt: product.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: product.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
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

	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("category = ?", category)
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
		items = append(items, toProductResponse(product))
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
