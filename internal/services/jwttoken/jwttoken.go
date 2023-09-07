package jwttoken

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	salt      = "nvbnergtefgsdfg314"
	secretKey = "asfdvcbiuyyrn#5241"
	tokenExp  = time.Hour * 3
)

type claims struct {
	jwt.RegisteredClaims
	UserID string
}

func Parse(accessToken string) (string, error) {
	claims := &claims{}

	token, err := jwt.ParseWithClaims(
		accessToken,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		},
	)

	if err != nil {
		return "", err
	}

	if !token.Valid || claims.UserID == "" {
		return "", fmt.Errorf("token is not valid")
	}

	return claims.UserID, nil
}

func Generate(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})

	accessToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return accessToken, nil
}
