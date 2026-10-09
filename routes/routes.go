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
			auth.GET("/me", handlers.AuthMiddleware(db), authHandler.Me)
			auth.POST("/logout", handlers.AuthMiddleware(db), authHandler.Logout)
		}

		productHandler := &handlers.ProductHandler{DB: db}
		r.POST("/products", handlers.AuthMiddleware(db), productHandler.Create)
		r.GET("/products", handlers.AuthMiddleware(db), productHandler.List)

		transactionHandler := &handlers.TransactionHandler{DB: db}
		authMiddleware := handlers.AuthMiddleware(db)
		r.POST("/sales", authMiddleware, transactionHandler.CreateSale)
		r.GET("/income", authMiddleware, transactionHandler.Income)
		r.GET("/income/export", authMiddleware, transactionHandler.ExportIncome)
		r.GET("/outcome", authMiddleware, transactionHandler.Outcome)
		r.GET("/outcome/export", authMiddleware, transactionHandler.ExportOutcome)
	}
}
