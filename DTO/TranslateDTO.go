package dto

type PendingTranslationItem struct {
	FolderID  int    `gorm:"column:folder_id" json:"folder_id"`
	Name      string `gorm:"column:name" json:"name"`
	Thumbnail string `gorm:"column:thumbnail" json:"thumbnail"`
}

type UpdateTranslationStatusRequest struct {
	Status       string `json:"status" binding:"required"`
	ErrorMessage string `json:"error_message"`
}
