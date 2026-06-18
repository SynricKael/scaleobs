package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims represents the JWT claims.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// contextKey is used for storing auth info in request context.
type contextKey string

const (
	// UserContextKey is the key for the authenticated user in request context.
	UserContextKey contextKey = "username"
)

// TokenExpiry is the default JWT token validity duration.
const TokenExpiry = 24 * time.Hour

// GenerateToken creates a JWT for the given username.
func GenerateToken(username string, secret string) (string, int64, error) {
	expiresAt := time.Now().Add(TokenExpiry)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "server-ops-portal",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}

	return tokenString, expiresAt.Unix(), nil
}

// ValidateToken parses and validates a JWT string.
func ValidateToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// HashPassword creates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a password against a bcrypt hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Middleware creates an HTTP middleware that validates JWT tokens.
// Only paths matching protectPaths require authentication; all other paths are public.
func Middleware(secret string, protectPaths []string) func(http.Handler) http.Handler {
	protectMap := make(map[string]bool, len(protectPaths))
	for _, p := range protectPaths {
		protectMap[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this path needs authentication
			needsAuth := false
			for protectPath := range protectMap {
				if strings.HasPrefix(r.URL.Path, protectPath) {
					needsAuth = true
					break
				}
			}

			if !needsAuth {
				next.ServeHTTP(w, r)
				return
			}

			// Extract Bearer token (from header or cookie)
			tokenString := ""

			// Try Authorization header first
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenString = parts[1]
				}
			}

			// Fall back to cookie (needed for iframe embeds)
			if tokenString == "" {
				if cookie, err := r.Cookie("token"); err == nil {
					tokenString = cookie.Value
				}
			}

			if tokenString == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			claims, err := ValidateToken(tokenString, secret)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"invalid token: %s"}`, err.Error()), http.StatusUnauthorized)
				return
			}

			// Store username in context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserContextKey, claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
