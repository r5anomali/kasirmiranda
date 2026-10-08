package handlers

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

func DBMiddleware(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
