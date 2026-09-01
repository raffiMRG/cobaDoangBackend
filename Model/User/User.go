package User

import "time"

type User struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	Username string    `gorm:"type:varchar(255);unique;not null" json:"username"`
	Password string    `gorm:"type:varchar(255);not null" json:"-"`
	CreateAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"create_at"`
}
