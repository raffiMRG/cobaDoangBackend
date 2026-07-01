package AuthRepositorys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	refreshTokenModel "web_backend/Model/RefreshToken"
	userModel "web_backend/Model/User"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

type accessTokenClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func jwtSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

func FindUserByUsername(db *gorm.DB, username string) (*userModel.User, error) {
	var user userModel.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func VerifyPassword(hashedPassword, plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}

func HashPassword(plainPassword string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func CreateUser(db *gorm.DB, username, plainPassword string) error {
	hashed, err := HashPassword(plainPassword)
	if err != nil {
		return err
	}
	user := userModel.User{Username: username, Password: hashed}
	return db.Create(&user).Error
}

// GenerateAccessToken issues a short-lived JWT carrying the user id.
func GenerateAccessToken(userID uint) (string, error) {
	claims := accessTokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ValidateAccessToken verifies signature+expiry and returns the embedded user id.
func ValidateAccessToken(tokenString string) (uint, error) {
	claims := &accessTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		return 0, errors.New("invalid or expired token")
	}
	return claims.UserID, nil
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateAndStoreRefreshToken creates a long-lived opaque refresh token,
// persisting only its hash so a leaked DB dump can't be replayed directly.
func GenerateAndStoreRefreshToken(db *gorm.DB, userID uint) (string, error) {
	rawToken, err := generateRandomToken()
	if err != nil {
		return "", err
	}
	refreshToken := refreshTokenModel.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
		Revoked:   false,
	}
	if err := db.Create(&refreshToken).Error; err != nil {
		return "", err
	}
	return rawToken, nil
}

// ValidateRefreshToken returns the associated user id if the token is
// known, not revoked, and not expired.
func ValidateRefreshToken(db *gorm.DB, rawToken string) (uint, error) {
	var refreshToken refreshTokenModel.RefreshToken
	err := db.Where("token_hash = ? AND revoked = ? AND expires_at > ?", hashToken(rawToken), false, time.Now()).
		First(&refreshToken).Error
	if err != nil {
		return 0, errors.New("invalid or expired refresh token")
	}
	return refreshToken.UserID, nil
}

func RevokeRefreshToken(db *gorm.DB, rawToken string) error {
	return db.Model(&refreshTokenModel.RefreshToken{}).
		Where("token_hash = ?", hashToken(rawToken)).
		Update("revoked", true).Error
}
