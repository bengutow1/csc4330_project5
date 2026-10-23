// Author: Ben Gutowski
package crew

import (
	"strings"

	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
)

// nameTag holds a crew's name to app.crew's check, 1 to 30 characters (keep the two in step), on
// one line of plain text.
type nameTag struct {
	Name string `pfield:"name" validate:"required,max=30,plaintext"`
}

// codeTag holds a code to app.crew's check: six digits.
type codeTag struct {
	Code string `pfield:"code" validate:"len=6,numeric"`
}

// buildName trims a crew's name, then validates it.
func buildName(raw string) (string, handlers.FieldErrors) {
	name := strings.TrimSpace(raw)
	invalid := handlers.Validate(nameTag{Name: name})
	if invalid != nil {
		return "", invalid
	}
	return name, nil
}

// buildCode trims a code, then validates it. Spaces a player typed between the digits go too.
func buildCode(raw string) (string, handlers.FieldErrors) {
	code := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	invalid := handlers.Validate(codeTag{Code: code})
	if invalid != nil {
		return "", invalid
	}
	return code, nil
}
