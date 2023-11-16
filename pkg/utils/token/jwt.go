package token

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

const title = "blade"

// Claims jwt object
type Claims struct {
	ID   uint
	Name string
	jwt.StandardClaims
}

// GenerateJWTToken 生成jwt格式token
func GenerateJWTToken(claims *Claims, exp int64) (string, int64, error) {
	expires := time.Now().Add(time.Second * time.Duration(exp)).Unix()
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString([]byte(title))
	return token, expires, err
}

// ParseJWTToken validate jwt token
func ParseJWTToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(title), nil
	})
	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}

	return nil, err
}
