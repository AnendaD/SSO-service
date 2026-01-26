package refreshtoken

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const tokenLength = 32

func NewRefreshToken() (string, error) {
	bytes := make([]byte, tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
