// Author: LAPTOP-BPMQJP10\andyk
package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// NewCode draws an unbiased six-digit invite, keeping leading zeroes. The insert owns uniqueness.
func NewCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("auth: generate invite code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
