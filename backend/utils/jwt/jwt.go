package jwt

import (
	"errors"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"
)

var secretKey = []byte("your-very-secret-key")

type Claims struct {
	UserId uint   `json:"user_id"`
	Role   string `json:"role"`
	jwtv4.RegisteredClaims
}

func GenerateSecureToken(userId uint, role string) (string, error) {
	claims := &Claims{
		UserId: userId,
		Role:   role,
		RegisteredClaims: jwtv4.RegisteredClaims{
			ExpiresAt: jwtv4.NewNumericDate(time.Now().Add(24 * time.Hour)), // Expiration time of 24 hours
		},
	}

	token := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ParseToken(tokenString string) (uint, string, error) {
	token, err := jwtv4.ParseWithClaims(tokenString, &Claims{}, func(token *jwtv4.Token) (interface{}, error) {
		// Ensure the token method is HMAC (HS256)
		if _, ok := token.Method.(*jwtv4.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return secretKey, nil
	})
	if err != nil {
		if ve, ok := err.(*jwtv4.ValidationError); ok {
			if ve.Errors&jwtv4.ValidationErrorExpired != 0 {
				return 0, "", errors.New("token is expired")
			}
		}
		return 0, "", errors.New("invalid token")
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserId, claims.Role, nil
	}
	return 0, "", errors.New("invalid token claims")
}
