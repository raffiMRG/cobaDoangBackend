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
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Thumbnail         string    `json:"thumbnail"`
	IsCompleted       bool      `json:"is_completed"`
	CreateAt          time.Time `json:"create_at"`
	Page              []string  `json:"page"`
	IsBookmarked      bool      `json:"is_bookmarked"`
	IsTranslated      bool      `json:"is_translated"`
	TranslationStatus string    `json:"translation_status"`
	TranslationError  string    `json:"translation_error,omitempty"`
}

type NewFolderQuery struct {
	ID           int       `gorm:"column:id" json:"id"`
	Name         string    `gorm:"column:name" json:"name"`
	Thumbnail    string    `gorm:"column:thumbnail" json:"thumbnail"`
	IsCompleted  bool      `gorm:"column:is_completed" json:"is_completed"`
	CreateAt     time.Time `gorm:"column:create_at" json:"create_at"`
	IsBookmarked bool      `gorm:"column:is_bookmarked" json:"is_bookmarked"`
	IsTranslated bool      `gorm:"column:is_translated" json:"is_translated"`
}
