package Bookmarkcontroller

import (
	"errors"
	"net/http"
	"strconv"
	"web_backend/Model/Bookmark"

	model "web_backend/Model"
	connection "web_backend/Model/Connection"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GET /bookmarks
func GetBookmarks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	db := connection.DB
	var bookmarks []Bookmark.BookmarkRes

	// Hitung offset
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Hitung total rows
	var total int64
	db.Table("bookmarks").Count(&total)

	if err := db.Table("bookmarks AS b").
		Select(`
		b.id AS bookmark_id,
		b.folder_id,
		b.created_at,
		nf.name AS folder_name,
		nf.thumbnail AS folder_thumbnail
	`).
		Joins("JOIN new_folders AS nf ON b.folder_id = nf.id").
		Order("b.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&bookmarks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Bungkus data + info pagination
	data := map[string]interface{}{
		"items": bookmarks,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": int((total + int64(limit) - 1) / int64(limit)), // total halaman
	}

	result := model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "Data retrieved successfully",
		Data:          data,
	}

	c.JSON(http.StatusOK, result)
}

// GET /bookmarks/:id
func GetBookmark(c *gin.Context) {
	db := connection.DB
	var bookmark Bookmark.Bookmark

	if err := db.Preload("Folder").First(&bookmark, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bookmark not found"})
		return
	}
	c.JSON(http.StatusOK, bookmark)
}

// POST /bookmarks
func ToggleBookmark(c *gin.Context) {
	db := connection.DB

	var input struct {
		FolderID uint `json:"folder_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing Bookmark.Bookmark
	err := db.Where("folder_id = ?", input.FolderID).First(&existing).Error

	if err == nil {
		// Jika bookmark sudah ada → hapus (unbookmark)
		if delErr := db.Delete(&existing).Error; delErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": delErr.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":       "Bookmark removed successfully",
			"is_bookmarked": false,
			"folder_id":     input.FolderID,
		})
		return
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// Jika belum ada → buat baru
		newBookmark := Bookmark.Bookmark{FolderID: input.FolderID}
		if createErr := db.Create(&newBookmark).Error; createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": createErr.Error()})
			return
		}

		// Kembalikan response sesuai format awal kamu
		c.JSON(http.StatusCreated, gin.H{
			"id":            newBookmark.ID,
			"folder_id":     newBookmark.FolderID,
			"is_bookmarked": true,
			"created_at":    newBookmark.CreatedAt,
		})
		return
	} else {
		// Error lain (selain "not found")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// POST /bookmarks
func CreateBookmark(c *gin.Context) {
	db := connection.DB
	var input struct {
		FolderID uint `json:"folder_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bookmark := Bookmark.Bookmark{FolderID: input.FolderID}
	if err := db.Create(&bookmark).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bookmark)
}

// DELETE /bookmarks/:id
func DeleteBookmark(c *gin.Context) {
	db := connection.DB
	if err := db.Delete(&Bookmark.Bookmark{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Bookmark deleted"})
}
