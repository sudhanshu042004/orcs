package config

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sudhanshu042004/orcs/types"
)

func CreateToken(id int64, email string) (string, error) {
	secreteByte := []byte(os.Getenv("JWT_SECRET"))

	//token generation
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":    id,
			"email": email,
			"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
		})

	// signed token
	tokenString, err := token.SignedString(secreteByte)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func VerifyToken(tokenString string) (types.JwtPayload, error) {
	secretKey := []byte(os.Getenv("JWT_SECRET"))

	//parse token
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return types.JwtPayload{}, err
	}
	if !token.Valid {
		return types.JwtPayload{}, fmt.Errorf("invalid token")
	}

	// claim email and id
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return types.JwtPayload{}, fmt.Errorf("invalid claims")
	}
	userEmail, ok := claims["email"].(string)
	if !ok {
		return types.JwtPayload{}, fmt.Errorf("email not found in token")
	}
	userId, ok := claims["id"].(int64)
	if !ok {
		return types.JwtPayload{}, fmt.Errorf("id not found in token")
	}

	return types.JwtPayload{Email: userEmail, Id: userId}, nil
}
