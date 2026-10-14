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
		productAuth := handlers.AuthMiddleware(db)
		r.POST("/products", productAuth, productHandler.Create)
		r.PATCH("/products/:id/stock", productAuth, productHandler.AddStock)
		r.GET("/products/search", productAuth, productHandler.Search)
		r.GET("/products", productAuth, productHandler.List)

		transactionHandler := &handlers.TransactionHandler{DB: db}
		authMiddleware := handlers.AuthMiddleware(db)
		r.POST("/sales", authMiddleware, transactionHandler.CreateSale)
		r.GET("/income", authMiddleware, transactionHandler.Income)
		r.GET("/income/export", authMiddleware, transactionHandler.ExportIncome)
		r.GET("/outcome", authMiddleware, transactionHandler.Outcome)
		r.GET("/outcome/export", authMiddleware, transactionHandler.ExportOutcome)
		r.POST("/outcome/:id/pay", authMiddleware, transactionHandler.PayDebt)
		r.POST("/loans", authMiddleware, transactionHandler.CreateLoan)
	}
}
