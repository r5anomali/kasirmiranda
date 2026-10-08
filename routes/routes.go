package routes

import (
	"kasirmiranda/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	r.GET("/health", handlers.Health)

	if db != nil {
		authHandler := &handlers.AuthHandler{DB: db}
		auth := r.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", handlers.AuthMiddleware(), authHandler.Me)
		}

		productHandler := &handlers.ProductHandler{DB: db}
		r.GET("/products", handlers.AuthMiddleware(), productHandler.List)
	}
}
