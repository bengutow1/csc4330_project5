// Author: Caden Lund
package handlers

import (
	"fmt"

	errorv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/error/v1"
)

// FieldError is one invalid request field. Field is the proto field name, which is what the
// client keys its inputs by; Message goes on screen verbatim, so it is written for the player.
type FieldError struct {
	Field   string
	Message string
}

// FieldErrors is every invalid field of one request. Validation collects them all instead of
// stopping at the first, so a form shows all its problems in one round trip.
type FieldErrors []FieldError

// Error leads with the first problem, which is what a client without field handling shows.
func (e FieldErrors) Error() string {
	switch len(e) {
	case 0:
		return "invalid request"
	case 1:
		return e[0].Message
	default:
		return fmt.Sprintf("%s (and %d more)", e[0].Message, len(e)-1)
	}
}

// detail is the wire form. Only ConnectErr builds it.
func (e FieldErrors) detail() *errorv1.BadRequest {
	violations := make([]*errorv1.FieldViolation, len(e))
	for i, fe := range e {
		violations[i] = &errorv1.FieldViolation{Field: fe.Field, Description: fe.Message}
	}
	return &errorv1.BadRequest{FieldViolations: violations}
}
