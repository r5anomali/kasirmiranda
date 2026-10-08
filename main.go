package main

import (
	"log"
	"os"

	"kasirmiranda/config"
	"kasirmiranda/models"
	"kasirmiranda/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	db, err := config.ConnectDatabase()
	if err != nil {
		log.Printf("Warning: Database connection failed (%v). Running without database features.", err)
	} else {
		log.Println("Database connected successfully!")
		if err := db.AutoMigrate(&models.User{}, &models.Product{}); err != nil {
			log.Fatalf("Database migration failed: %v", err)
		}
	}

	routes.RegisterRoutes(r, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
