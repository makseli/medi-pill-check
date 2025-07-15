package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/makseli/medi-pill-check/internal/config"
	"github.com/makseli/medi-pill-check/internal/models"
)

type Handlers struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Handlers {
	return &Handlers{
		db:  db,
		cfg: cfg,
	}
}

// Kullanıcı kaydı
func (h *Handlers) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=32"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := models.User{
		Username: req.Username,
		Email:    strings.ToLower(req.Email),
		Password: string(hash),
	}
	if err := h.db.Create(&user).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User with this email or username already exists"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create user"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "username": user.Username, "email": user.Email})
}

// Kullanıcı girişi
func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := h.db.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	token := generateJWT(user.ID, h.cfg.JWTSecret)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Kullanıcıları listele
func (h *Handlers) ListUsers(c *gin.Context) {
	var users []models.User
	h.db.Find(&users)
	c.JSON(http.StatusOK, users)
}

// Kullanıcıyı getir
func (h *Handlers) GetUser(c *gin.Context) {
	var user models.User
	if err := h.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// Kullanıcıyı güncelle
func (h *Handlers) UpdateUser(c *gin.Context) {
	var user models.User
	if err := h.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Username != "" && req.Username != user.Username {
		var count int64
		h.db.Model(&models.User{}).Where("username = ? AND id != ?", req.Username, user.ID).Count(&count)
		if count > 0 {
			c.JSON(400, gin.H{"error": "Username is already taken"})
			return
		}
		user.Username = req.Username
	}
	if req.Email != "" && req.Email != user.Email {
		var count int64
		h.db.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, user.ID).Count(&count)
		if count > 0 {
			c.JSON(400, gin.H{"error": "Email is already taken"})
			return
		}
		user.Email = req.Email
	}
	h.db.Save(&user)
	c.JSON(http.StatusOK, user)
}

// Kullanıcıyı sil (soft delete)
func (h *Handlers) DeleteUser(c *gin.Context) {
	if err := h.db.Delete(&models.User{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Health status endpoint
func (h *Handlers) Health(c *gin.Context) {

	// TODO: Check if the database & redis is connected

	c.JSON(200, gin.H{
		"status":  "ok",
		"message": "MedipillCheck API is running",
	})
}

// JWT token üretimi (config'den secret alır, 1 saatlik süre)
func generateJWT(userID uint, secret string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}
