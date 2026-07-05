package Translation

import (
	"time"
	"web_backend/Model/NewFolder"
)

type Translation struct {
	ID        uint                `json:"id" gorm:"primaryKey"`
	FolderID  uint                `json:"folder_id"`
	Folder    NewFolder.NewFolder `json:"folder" gorm:"foreignKey:FolderID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time           `json:"created_at"`
}
