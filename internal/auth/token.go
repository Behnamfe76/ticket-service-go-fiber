package auth

import (
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// TokenManager handles issuing and validating JWT tokens.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager builds a new manager.
func NewTokenManager(secret string, ttlMinutes int) *TokenManager {
	if ttlMinutes <= 0 {
		ttlMinutes = 60
	}
	return &TokenManager{secret: []byte(secret), ttl: time.Duration(ttlMinutes) * time.Minute}
}

// Claims describes JWT payload.
type Claims struct {
	SubjectID string             `json:"sub"`
	Subject   domain.SubjectType `json:"subject"`
	Role      *domain.StaffRole  `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken builds and signs a JWT for the subject.
func (tm *TokenManager) GenerateToken(subjectID string, subject domain.SubjectType, role *domain.StaffRole) (string, time.Time, error) {
	expiresAt := time.Now().Add(tm.ttl)
	claims := &Claims{
		SubjectID: subjectID,
		Subject:   subject,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subjectID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenString, expiresAt, nil
}

// ParseToken validates and returns claims.
func (tm *TokenManager) ParseToken(tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return tm.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
