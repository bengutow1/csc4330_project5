// Author: LAPTOP-BPMQJP10\andyk
package auth_test

import (
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/auth"
)

// TestInviteCodesAreAlwaysSixASCIIDigits prevents lost leading zeroes or non-digit invites.
func TestInviteCodesAreAlwaysSixASCIIDigits(t *testing.T) {
	t.Parallel()
	for range 100 {
		code, err := auth.NewCode()
		if err != nil {
			t.Fatalf("code: %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("length = %d, want 6", len(code))
		}
		for _, digit := range code {
			if digit < '0' || digit > '9' {
				t.Fatalf("non-ASCII digit in code")
			}
		}
	}
}
