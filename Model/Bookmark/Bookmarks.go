package Bookmark

import (
	"time"
	"web_backend/Model/NewFolder"
)

type Bookmark struct {
	ID        uint                `json:"id" gorm:"primaryKey"`
	FolderID  uint                `json:"folder_id"`
	Folder    NewFolder.NewFolder `json:"folder" gorm:"foreignKey:FolderID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time           `json:"created_at"`
}

type BookmarkRes struct {
	BookmarkID      uint      `json:"bookmark_id" gorm:"primaryKey"`
	FolderID        uint      `json:"folder_id"`
	CreatedAt       time.Time `json:"created_at"`
	FolderName      string    `json:"folder_name"`
	FolderThumbnail string    `json:"folder_thumbnail"`
}
