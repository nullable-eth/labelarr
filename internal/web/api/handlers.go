package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/optikshell/plex-label-manager/internal/models"
)

type Handler struct {
	DB *gorm.DB
}

// Search media in Plex library
func (h *Handler) SearchMedia(c *gin.Context) {
	query := c.Query("q")
	mediaType := c.Query("type") // "movie", "tv", or "all"

	var media []models.Media
	db := h.DB.Preload("Labels")

	if query != "" {
		db = db.Where("title ILIKE ?", "%"+query+"%")
	}

	if mediaType != "" && mediaType != "all" {
		db = db.Where("type = ?", mediaType)
	}

	if err := db.Find(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, media)
}

// Get media by ID with labels
func (h *Handler) GetMedia(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	var media models.Media
	if err := h.DB.Preload("Labels").First(&media, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	c.JSON(http.StatusOK, media)
}

// Update media labels
func (h *Handler) UpdateMediaLabels(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	var request struct {
		LabelIDs []uint `json:"label_ids"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var media models.Media
	if err := h.DB.First(&media, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Get new labels
	var labels []models.Label
	if err := h.DB.Find(&labels, request.LabelIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Replace associations
	if err := h.DB.Model(&media).Association("Labels").Replace(labels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also update Plex labels
	go h.syncLabelsWithPlex(&media, labels)

	c.JSON(http.StatusOK, gin.H{"message": "Labels updated successfully"})
}

// Get all labels
func (h *Handler) GetLabels(c *gin.Context) {
	var labels []models.Label
	if err := h.DB.Find(&labels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, labels)
}

// Create new label
func (h *Handler) CreateLabel(c *gin.Context) {
	var label models.Label
	if err := c.ShouldBindJSON(&label); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.Create(&label).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, label)
}

// Filter media by labels
func (h *Handler) FilterByLabels(c *gin.Context) {
	labelNames := c.QueryArray("labels")

	var media []models.Media
	query := h.DB.Preload("Labels")

	if len(labelNames) > 0 {
		query = query.Joins("JOIN media_labels ON media.id = media_labels.media_id").
			Joins("JOIN labels ON media_labels.label_id = labels.id").
			Where("labels.name IN ?", labelNames).
			Group("media.id").
			Having("COUNT(DISTINCT labels.id) = ?", len(labelNames))
	}

	if err := query.Find(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, media)
}
