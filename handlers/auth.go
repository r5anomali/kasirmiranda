package handlers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"kasirmiranda/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB *gorm.DB
}

type loginRequest struct {
	Username string `json:"username" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve user"})
		return
	}
	//keren

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"email": user.Username,
		"roles": []string{user.Role},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input loginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username or password is invalid"})
		return
	}

	var user models.User
	err := h.DB.Where("username = ?", strings.TrimSpace(input.Username)).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not authenticate"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication is not configured"})
		return
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(24 * time.Hour).Unix(),
	})
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": signedToken, "token_type": "Bearer", "expires_in": 86400})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
		return
	}

	tokenString := strings.TrimPrefix(header, "Bearer ")
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication is not configured"})
		return
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	expValue, ok := claims["exp"]
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token expiration"})
		return
	}

	var expiresAt time.Time
	switch value := expValue.(type) {
	case float64:
		expiresAt = time.Unix(int64(value), 0)
	case int64:
		expiresAt = time.Unix(value, 0)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token expiration"})
		return
	}

	signature := hashToken(tokenString)
	if err := h.DB.Create(&models.TokenBlacklist{Signature: signature, ExpiresAt: expiresAt}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
