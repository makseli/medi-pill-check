package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/makseli/medi-pill-check/internal/models"
	"gorm.io/gorm"
)

type MedicineHandler struct {
	db *gorm.DB
}

func NewMedicineHandler(db *gorm.DB) *MedicineHandler {
	return &MedicineHandler{db: db}
}

func (h *MedicineHandler) Create(c *gin.Context) {
	var req struct {
		Type         int    `json:"type" binding:"required,oneof=1 2 3"`
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
		Type:         req.Type,
		Name:         req.Name,
		Dose:         req.Dose,
		ScheduleType: req.ScheduleType,
		Description:  req.Description,
	}
	if err := h.db.Create(&med).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create medication"})
		return
	}
	logAuditWithContext(c, h.db, &userID, "medication_create", "Medication created")
	c.JSON(http.StatusCreated, med)
}

func (h *MedicineHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	var meds []models.Medication
	h.db.Where("user_id = ?", userID).Find(&meds)
	c.JSON(http.StatusOK, meds)
}

func (h *MedicineHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	var med models.Medication
	if err := h.db.Where("user_id = ? AND id = ?", userID, c.Param("id")).First(&med).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Medication not found"})
		return
	}
	c.JSON(http.StatusOK, med)
}

func (h *MedicineHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	var med models.Medication
	if err := h.db.Where("user_id = ? AND id = ?", userID, c.Param("id")).First(&med).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Medication not found"})
		return
	}
	var req struct {
		Type         int    `json:"type"`
		Name         string `json:"name"`
		Dose         string `json:"dose"`
		ScheduleType string `json:"schedule_type"`
		Description  string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type != 0 {
		med.Type = req.Type
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
	logAuditWithContext(c, h.db, &userID, "medication_update", "Medication updated")
	c.JSON(http.StatusOK, med)
}

func (h *MedicineHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	h.db.Where("user_id = ? AND id = ?", userID, c.Param("id")).Delete(&models.Medication{})
	logAuditWithContext(c, h.db, &userID, "medication_delete", "Medication deleted")
	c.Status(http.StatusNoContent)
}
