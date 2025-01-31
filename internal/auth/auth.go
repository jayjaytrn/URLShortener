package auth

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
)

const SecretKey = "supersecretkey"

type Manager struct{}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) BuildJWTStringWithNewID(userID string) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

func (m *Manager) GetUserIDFromJWTString(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SecretKey), nil
		})
	if err != nil {
		return "", fmt.Errorf("token error: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("token is not valid: %w", err)
	}

	if claims.UserID == "" {
		return "", fmt.Errorf("token is valid but userID is missing")
	}

	fmt.Println("Token is valid: " + claims.UserID)
	return claims.UserID, nil
}
