// Author: LAPTOP-BPMQJP10\andyk
package combo

import (
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
)

// validateCode mirrors app.combo's ASCII six-digit check.
func validateCode(code string) error {
	if len(code) != 6 {
		return handlers.FieldErrors{{Field: "code", Message: "enter the six-digit code"}}
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return handlers.FieldErrors{{Field: "code", Message: "enter the six-digit code"}}
		}
	}
	return nil
}
