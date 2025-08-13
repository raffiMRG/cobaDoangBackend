package NewFolder

import "time"

type NewFolder struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	IsCompleted bool      `gorm:"type:boolean;" json:"is_completed"`
	CreateAt    time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"create_at"`
}

// func (NewFolder) TableName() string {
// 	return "new_folder" // Nama tabel di database
// }
