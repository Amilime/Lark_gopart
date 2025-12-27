package ws

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
)

// 必须和 Java 端的 application.yml 保持完全一致！
// 如果那边改了，这里也要改
var jwtSecret = []byte("This_Is_A_Super_Secure_Key_For_Lark_2025")

// 定义载荷结构 (对应 Java 里的 Claims)
type MyClaims struct {
	Uid  int64  `json:"uid"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// ParseToken 验证并解析 Token
func ParseToken(tokenString string) (*MyClaims, error) {
	// 1. 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// 2. 验证有效性
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
