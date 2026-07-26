package BugReport

import (
	"time"
	"web_backend/Model/NewFolder"
)

type BugReport struct {
	ID          uint                `json:"id" gorm:"primaryKey"`
	FolderID    uint                `json:"folder_id"`
	Folder      NewFolder.NewFolder `json:"folder" gorm:"foreignKey:FolderID;constraint:OnDelete:CASCADE"`
	Description string              `json:"description"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
