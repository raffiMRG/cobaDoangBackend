package dto

type InputDataRes struct {
	// ID     int    `json:"Id"`
	ID uint `gorm:"primaryKey" json:"id"`
	// ID     int    `gorm:"primaryKey" json:"id"`
	Title  string `json:"Title"`
	Status bool   `json:"Status"`
}
