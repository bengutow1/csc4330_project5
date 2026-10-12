// Author: Caden Lund
package handlers

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// validate is built once and is safe for concurrent use. A field reports under its `pfield` tag,
// the proto field name, because that is the name the client keys its inputs by.
var validate = sync.OnceValue(func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		return f.Tag.Get("pfield")
	})
	return v
})

// Validate checks a struct against its `validate` tags and returns one FieldError per invalid
// field in declaration order, or nil when everything is valid. Describe the request at the call
// site as an anonymous struct:
//
//	invalid := handlers.Validate(struct {
//		Code string `pfield:"code" validate:"len=6"`
//	}{code})
//	if invalid != nil {
//		return nil, h.fail(op, invalid)
//	}
func Validate(req any) FieldErrors {
	err := validate().Struct(req)
	if err == nil {
		return nil
	}
	var violations validator.ValidationErrors
	if !errors.As(err, &violations) {
		// Not a struct: a bug at the call site. Refuse rather than wave the request through, so
		// that RPC's own tests fail on it.
		return FieldErrors{{Field: "", Message: "invalid request"}}
	}
	out := make(FieldErrors, 0, len(violations))
	for _, fe := range violations {
		out = append(out, FieldError{Field: fe.Field(), Message: sentence(fe)})
	}
	return out
}

// sentence turns the failing rule into what the player reads under the input. A rule with no
// case here still refuses, as "is invalid"; add a case when that reads badly.
func sentence(fe validator.FieldError) string {
	label := strings.ReplaceAll(fe.Field(), "_", " ")
	// Bounds on text and lists count runes and items; numbers compare by value and need no unit.
	unit := ""
	switch fe.Kind() {
	case reflect.String:
		unit = " characters"
	case reflect.Slice, reflect.Array, reflect.Map:
		unit = " items"
	}

	switch fe.Tag() {
	case "required":
		return label + " is required"
	case "gt":
		// gt=0 is how an id says "required".
		if fe.Param() == "0" {
			return label + " is required"
		}
		return label + " must be more than " + fe.Param() + unit
	case "gte", "min":
		return label + " must be at least " + fe.Param() + unit
	case "lt":
		return label + " must be less than " + fe.Param() + unit
	case "lte", "max":
		return label + " must be at most " + fe.Param() + unit
	case "len":
		return label + " must be exactly " + fe.Param() + unit
	case "oneof":
		return label + " must be " + strings.ReplaceAll(fe.Param(), " ", " or ")
	default:
		return label + " is invalid"
	}
}
