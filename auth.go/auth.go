package auth

import (
	"errors"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidJWT        = errors.New("invalid or unauthorized jwt")
)

// VerifyJWTFromHeader parses and verifies a JWT using the Authorization header and public key
func VerifyJWTFromHeader(header string, pubKeyPath string) error {
	if header == "" {
		return ErrMissingAuthHeader
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ErrInvalidJWT
	}

	token := parts[1]

	pub, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return err
	}

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM(pub)
	})
	if err != nil || !parsedToken.Valid {
		return ErrInvalidJWT
	}

	return nil
}
