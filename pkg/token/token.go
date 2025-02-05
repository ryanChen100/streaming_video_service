package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RoleType set member role
type RoleType string

const (
	// RoleAdmin is the admin role
	RoleAdmin RoleType = "admin"
	// RoleUser is the user role
	RoleUser RoleType = "user"
	// RoleMember is the member role
	RoleMember RoleType = "member"
	// RoleGuest is the guest role
	RoleGuest RoleType = "guest"
)

// Claims structure for custom claims in JWT
type Claims struct {
	MemberID string `json:"user_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Secret Key for JWT signing and validation
var (
	JWTSecret       = []byte("secure_secret_key")
	tokenExpiration = 60 * time.Minute
)

// GenerateJWT generates a JWT token
func GenerateJWT(memberID, role, issuer string) (string, error) {
	claims := Claims{
		MemberID: memberID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ParseJWT parses a JWT and extracts the Claims
func ParseJWT(tokenStr string) (*Claims, error) {
	// Parse the token and extract the claims
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Check if the signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return JWTSecret, nil
	})

	if err != nil {
		// Handle other parsing errors (e.g. invalid signature, claims invalid, etc.)
		return nil, err
	}

	// Extract claims from the token
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// CheckJWTNotExpire check JWT token not expires
func CheckJWTNotExpire(t string) (bool, error) {
	// Get token from Authorization header
	if len(t) < 7 || t[:7] != "Bearer " {
		return true, errors.New("Invalid or missing token")
	}

	tokenStr := t[7:]

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return JWTSecret, nil
	})

	if err != nil {
		return true, err
	}

	tokenExpire, err := token.Claims.GetExpirationTime()
	if err != nil {
		return true, nil
	}

	return tokenExpire.After(time.Now()), nil
}

// DeleteJWT delete JWT token
func DeleteJWT(t string) (bool, error) {
	tokenStr := t[7:]

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return JWTSecret, nil
	})

	if err != nil {
		return true, err
	}

	tokenExpire, err := token.Claims.GetExpirationTime()
	if err != nil {
		return true, nil
	}

	return tokenExpire.After(time.Now()), nil
}
