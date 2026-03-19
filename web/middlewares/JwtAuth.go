package middlewares

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"inovasiriset.co.id/docker/manager/conf"
)

type JWTClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func JwtGenerateToken(username string, maxAge time.Duration) (string, error) {
	claims := &JWTClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(maxAge)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "mandok",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(conf.AppConfig.JWTSecret))
}

func JwtValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.AppConfig.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
