package RefreshToken

import "time"

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	TokenHash string    `gorm:"type:varchar(255);not null" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Revoked   bool      `gorm:"not null;default:false" json:"revoked"`
	CreateAt  time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"create_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
