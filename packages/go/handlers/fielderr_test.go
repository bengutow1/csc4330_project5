// Author: Caden Lund
package handlers_test

import (
	"testing"

	"github.com/cadenlund/fries-in-the-bag/packages/go/handlers"
)

// TestFieldErrorsReadAsTheirFirstProblem: the error's text is what a client without field
// handling shows, so it leads with the first problem and counts the rest.
func TestFieldErrorsReadAsTheirFirstProblem(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		errs handlers.FieldErrors
		want string
	}{
		{name: "none", errs: nil, want: "invalid request"},
		{
			name: "one",
			errs: handlers.FieldErrors{{Field: "handle", Message: "handle is required"}},
			want: "handle is required",
		},
		{
			name: "several",
			errs: handlers.FieldErrors{
				{Field: "handle", Message: "handle is required"},
				{Field: "avatar", Message: "avatar is required"},
				{Field: "roast_level", Message: "roast level must be mild or medium or extra_crispy"},
			},
			want: "handle is required (and 2 more)",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := c.errs.Error()

			if got != c.want {
				t.Errorf("Error() = %q, want %q", got, c.want)
			}
		})
	}
}
