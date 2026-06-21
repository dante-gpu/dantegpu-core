package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// I need a type for my context key to avoid collisions.
type contextKey string

// ContextKeyClaims is the key used to store JWT claims in the request context.
const ContextKeyClaims contextKey = "claims"

// Claims defines the structure of the JWT claims.
// I need to include standard claims and custom ones like Role and UserID.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a new JWT token for a given user.
// It uses the secret key and expiration duration from the configuration.
func GenerateJWT(user *User, secretKey string, expiration time.Duration) (string, time.Time, error) {
	// I should define the expiration time for the token.
	expirationTime := time.Now().Add(expiration)

	// I need to create the claims, including custom fields and standard ones.
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	// I need to create the token using the HS256 signing method and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// I should sign the token with the secret key.
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}

// ValidateJWT validates the given JWT token string.
// It returns the claims if the token is valid, otherwise returns an error.
func ValidateJWT(tokenString string, secretKey string) (*Claims, error) {
	claims := &Claims{}

	// I need to parse the token with the claims structure.
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// I must check the signing method!
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err // This handles expired tokens, invalid signatures, etc.
	}

	// I should check if the token is valid.
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
