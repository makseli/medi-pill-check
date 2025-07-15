package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/makseli/medi-pill-check/internal/config"
	"github.com/makseli/medi-pill-check/internal/database"
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
	logAuditWithContext(c, h.db, &user.ID, "register", "User registered")
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
	lockKey := "lock:" + strings.ToLower(req.Email)
	ctx := context.Background()
	locked, _ := database.RedisClient.Get(ctx, lockKey).Result()
	if locked == "1" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is locked due to too many failed login attempts. Please try again later."})
		return
	}
	var user models.User
	if err := h.db.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		incrementLoginAttempt(req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		incrementLoginAttempt(req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	resetLoginAttempt(req.Email)
	token := generateJWT(user.ID, h.cfg.JWTSecret)
	c.JSON(http.StatusOK, gin.H{"token": token})
	logAuditWithContext(c, h.db, &user.ID, "login", "User logged in")
}

func incrementLoginAttempt(email string) {
	ctx := context.Background()
	key := "login_attempt:" + strings.ToLower(email)
	cnt, _ := database.RedisClient.Incr(ctx, key).Result()
	if cnt == 1 {
		database.RedisClient.Expire(ctx, key, 15*time.Minute)
	}
	if cnt >= 5 {
		lockKey := "lock:" + strings.ToLower(email)
		database.RedisClient.Set(ctx, lockKey, "1", 15*time.Minute)
	}
}

func resetLoginAttempt(email string) {
	ctx := context.Background()
	key := "login_attempt:" + strings.ToLower(email)
	database.RedisClient.Del(ctx, key)
	lockKey := "lock:" + strings.ToLower(email)
	database.RedisClient.Del(ctx, lockKey)
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
		Password string `json:"password"`
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
	var newToken string
	if req.Password != "" {
		now := time.Now().UTC()
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user.Password = string(hash)
		user.PasswordChangedAt = &now
		newToken = generateJWTWithIat(user.ID, h.cfg.JWTSecret, now)
		logAuditWithContext(c, h.db, &user.ID, "password_change", "User changed password")
	}
	h.db.Save(&user)
	if newToken != "" {
		c.JSON(http.StatusOK, gin.H{"user": gin.H{"id": user.ID, "username": user.Username, "email": user.Email}, "token": newToken, "message": "Password changed, please use the new token."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "email": user.Email})
}

// Kullanıcıyı sil (soft delete)
func (h *Handlers) DeleteUser(c *gin.Context) {
	if err := h.db.Delete(&models.User{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Medication CRUD
func (h *Handlers) CreateMedication(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Dose         string `json:"dose" binding:"required"`
		ScheduleType string `json:"schedule_type" binding:"required,oneof=hourly daily weekly monthly"`
		Description  string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetUint("user_id")
	med := models.Medication{
		UserID:       userID,
		Name:         req.Name,
		Dose:         req.Dose,
		ScheduleType: req.ScheduleType,
		Description:  req.Description,
	}
	if err := h.db.Create(&med).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create medication"})
		return
	}
	c.JSON(http.StatusCreated, med)
}

func (h *Handlers) ListMedications(c *gin.Context) {
	userID := c.GetUint("user_id")
	var meds []models.Medication
	h.db.Where("user_id = ?", userID).Find(&meds)
	c.JSON(http.StatusOK, meds)
}

func (h *Handlers) GetMedication(c *gin.Context) {
	userID := c.GetUint("user_id")
	var med models.Medication
	if err := h.db.Where("user_id = ? AND id = ?", userID, c.Param("id")).First(&med).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Medication not found"})
		return
	}
	c.JSON(http.StatusOK, med)
}

func (h *Handlers) UpdateMedication(c *gin.Context) {
	userID := c.GetUint("user_id")
	var med models.Medication
	if err := h.db.Where("user_id = ? AND id = ?", userID, c.Param("id")).First(&med).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Medication not found"})
		return
	}
	var req struct {
		Name         string `json:"name"`
		Dose         string `json:"dose"`
		ScheduleType string `json:"schedule_type"`
		Description  string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name != "" {
		med.Name = req.Name
	}
	if req.Dose != "" {
		med.Dose = req.Dose
	}
	if req.ScheduleType != "" {
		med.ScheduleType = req.ScheduleType
	}
	if req.Description != "" {
		med.Description = req.Description
	}
	h.db.Save(&med)
	c.JSON(http.StatusOK, med)
}

func (h *Handlers) DeleteMedication(c *gin.Context) {
	userID := c.GetUint("user_id")
	if err := h.db.Where("user_id = ? AND id = ?", userID, c.Param("id")).Delete(&models.Medication{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete medication"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Health status endpoint
func (h *Handlers) Health(c *gin.Context) {
	// Veritabanı bağlantı kontrolü
	dbStatus := "ok"
	if h.db == nil {
		dbStatus = "not_connected"
	} else {
		dbErr := h.db.Exec("SELECT 1").Error
		if dbErr != nil {
			dbStatus = "error: " + dbErr.Error()
		}
	}

	// Redis bağlantı kontrolü
	redisStatus := "ok"
	if database.RedisClient == nil {
		redisStatus = "not_connected"
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := database.RedisClient.Ping(ctx).Err(); err != nil {
			redisStatus = "error: " + err.Error()
		}
	}

	c.JSON(200, gin.H{
		"status":   "ok",
		"message":  "MedipillCheck API is running",
		"database": dbStatus,
		"redis":    redisStatus,
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

// JWT token üretimi (config'den secret alır, 1 saatlik süre, iat desteği)
func generateJWTWithIat(userID uint, secret string, iat time.Time) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     iat.Add(1 * time.Hour).Unix(),
		"iat":     iat.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

// Refresh token üretimi
func generateRefreshToken(userID uint, secret string) (string, error) {
	exp := time.Now().Add(7 * 24 * time.Hour) // 7 gün
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     exp.Unix(),
		"type":    "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Refresh token'ı Redis'te blacklist et
func blacklistRefreshToken(token string, exp time.Time) error {
	ctx := context.Background()
	return database.RedisClient.Set(ctx, "blrt:"+token, "1", time.Until(exp)).Err()
}

// Refresh token blacklist kontrolü
func isRefreshTokenBlacklisted(token string) bool {
	ctx := context.Background()
	res, _ := database.RedisClient.Get(ctx, "blrt:"+token).Result()
	return res == "1"
}

func logAudit(db *gorm.DB, userID *uint, action, detail string) {
	db.Create(&models.AuditLog{
		UserID:    userID,
		Action:    action,
		Detail:    detail,
		CreatedAt: time.Now().UTC(),
	})
}

func logAuditWithContext(c *gin.Context, db *gorm.DB, userID *uint, action, detail string) {
	db.Create(&models.AuditLog{
		UserID:    userID,
		Action:    action,
		Detail:    detail,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		CreatedAt: time.Now().UTC(),
	})
}

func (h *Handlers) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
		return
	}
	// Token decode
	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}
	exp, _ := claims["exp"].(float64)
	if err := blacklistRefreshToken(req.RefreshToken, time.Unix(int64(exp), 0)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to blacklist token"})
		return
	}
	userID := c.GetUint("user_id")
	logAuditWithContext(c, h.db, &userID, "logout", "User logged out and refresh token blacklisted")
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *Handlers) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
		return
	}
	if isRefreshTokenBlacklisted(req.RefreshToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token is blacklisted"})
		return
	}
	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}
	userID, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user in token"})
		return
	}
	accessToken := generateJWT(uint(userID), h.cfg.JWTSecret)
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}
