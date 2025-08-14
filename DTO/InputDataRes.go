package dto

import "time"

type InputDataRes struct {
	// ID     int    `json:"Id"`
	ID uint `gorm:"primaryKey" json:"id"`
	// ID     int    `gorm:"primaryKey" json:"id"`
	Title  string `json:"Title"`
	Status bool   `json:"Status"`
}

type NewFolderResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Thumbnail   string    `json:"thumbnail"`
	IsCompleted bool      `json:"is_completed"`
	CreateAt    time.Time `json:"create_at"`
	Page        []string  `json:"page"`
}
