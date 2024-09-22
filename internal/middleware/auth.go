package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtSecret = []byte("your_secret_key")

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// 生成 JWT
func GenerateJWT(userID uint) (string, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 验证 JWT
func ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, errors.New("invalid token signature")
		}
		return nil, errors.New("invalid token")
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// 提取并验证 Authorization 头中的 JWT
func ExtractJWTFromHeader(r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("no Authorization header provided")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	return ValidateJWT(token)
}

// JWT 中间件
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ExtractJWTFromHeader(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// 添加用户 ID 到上下文中
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextKey("user_id"), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// 验证 token 并返回用户 ID
func ValidateToken(token string) (uint, error) {
	claims, err := ValidateJWT(token)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// 用于从上下文中获取用户 ID 的键
type contextKey string

const UserIDKey contextKey = "user_id"

// 从上下文中获取用户 ID
func GetUserIDFromContext(ctx context.Context) (uint, error) {
	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		return 0, errors.New("user ID not found in context")
	}
	return userID, nil
}
