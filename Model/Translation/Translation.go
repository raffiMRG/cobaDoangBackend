package Translation

import (
	"time"
	"web_backend/Model/NewFolder"
)

type Translation struct {
	ID           uint                `json:"id" gorm:"primaryKey"`
	FolderID     uint                `json:"folder_id"`
	Folder       NewFolder.NewFolder `json:"folder" gorm:"foreignKey:FolderID;constraint:OnDelete:CASCADE"`
	Status       string              `json:"status"`
	ErrorMessage string              `json:"error_message"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}
