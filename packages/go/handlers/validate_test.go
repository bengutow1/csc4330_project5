// Author: Caden Lund
package handlers_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
)

// TestValidateReportsEveryViolationInDeclarationOrder: one round trip shows the whole form's
// problems, under the proto names, in the order the form lists them.
func TestValidateReportsEveryViolationInDeclarationOrder(t *testing.T) {
	t.Parallel()

	got := handlers.Validate(struct {
		Code   string `pfield:"code" validate:"len=6"`
		Handle string `pfield:"handle" validate:"required,max=20"`
		ScanID int64  `pfield:"scan_id" validate:"gt=0"`
	}{Code: "48291", Handle: "", ScanID: 0})

	want := handlers.FieldErrors{
		{Field: "code", Message: "code must be exactly 6 characters"},
		{Field: "handle", Message: "handle is required"},
		{Field: "scan_id", Message: "scan id is required"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestValidateReturnsNilForAValidRequest: callers check `invalid != nil`, so an empty but non-nil
// result would refuse a good request.
func TestValidateReturnsNilForAValidRequest(t *testing.T) {
	t.Parallel()

	got := handlers.Validate(struct {
		Code   string `pfield:"code" validate:"len=6"`
		Handle string `pfield:"handle" validate:"required,max=20"`
	}{Code: "482913", Handle: "chef_maya"})

	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// TestValidateCountsLengthsInRunes: an ingredient like jalapeño must not hit a byte limit early.
func TestValidateCountsLengthsInRunes(t *testing.T) {
	t.Parallel()

	got := handlers.Validate(struct {
		Name string `pfield:"name" validate:"max=8"`
	}{Name: "jalapeño"}) // 8 runes, 9 bytes

	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// TestValidateWritesASentenceForEachRule: these messages go on screen verbatim, under the input.
func TestValidateWritesASentenceForEachRule(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		req  any
		want string
	}{
		{
			name: "required",
			req: struct {
				V string `pfield:"handle" validate:"required"`
			}{},
			want: "handle is required",
		},
		{
			name: "an id of zero",
			req: struct {
				V int64 `pfield:"combo_id" validate:"gt=0"`
			}{},
			want: "combo id is required",
		},
		{
			name: "more than",
			req: struct {
				V int32 `pfield:"points" validate:"gt=5"`
			}{V: 5},
			want: "points must be more than 5",
		},
		{
			name: "too short",
			req: struct {
				V string `pfield:"handle" validate:"min=2"`
			}{V: "a"},
			want: "handle must be at least 2 characters",
		},
		{
			name: "too long",
			req: struct {
				V string `pfield:"handle" validate:"max=3"`
			}{V: "abcd"},
			want: "handle must be at most 3 characters",
		},
		{
			name: "below a minimum",
			req: struct {
				V int32 `pfield:"limit" validate:"gte=1"`
			}{V: 0},
			want: "limit must be at least 1",
		},
		{
			name: "at an exclusive bound",
			req: struct {
				V int32 `pfield:"players" validate:"lt=5"`
			}{V: 5},
			want: "players must be less than 5",
		},
		{
			name: "over a maximum",
			req: struct {
				V int32 `pfield:"max_minutes" validate:"lte=45"`
			}{V: 46},
			want: "max minutes must be at most 45",
		},
		{
			name: "exact length",
			req: struct {
				V string `pfield:"code" validate:"len=6"`
			}{V: "123"},
			want: "code must be exactly 6 characters",
		},
		{
			name: "too many items",
			req: struct {
				V []string `pfield:"photos" validate:"max=3"`
			}{V: []string{"a", "b", "c", "d"}},
			want: "photos must be at most 3 items",
		},
		{
			name: "outside a vocabulary",
			req: struct {
				V string `pfield:"kind" validate:"oneof=fridge freezer pantry"`
			}{V: "garage"},
			want: "kind must be fridge or freezer or pantry",
		},
		{
			name: "a rule with no sentence of its own",
			req: struct {
				V string `pfield:"website" validate:"url"`
			}{V: "not a link"},
			want: "website is invalid",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := handlers.Validate(c.req)

			if len(got) != 1 {
				t.Fatalf("got %d violations (%v), want 1", len(got), got)
			}
			if got[0].Message != c.want {
				t.Errorf("message = %q, want %q", got[0].Message, c.want)
			}
		})
	}
}

// TestValidateRefusesSomethingThatIsNotAStruct: a call-site bug must fail closed, not wave the
// request through.
func TestValidateRefusesSomethingThatIsNotAStruct(t *testing.T) {
	t.Parallel()

	got := handlers.Validate("482913")

	if got == nil {
		t.Error("a string passed validation as if it were a valid request")
	}
}

// TestTheHandleTagHoldsTheHandleRule: it restates the handle check on app.account, so a bad handle
// becomes a message under the input instead of a redacted internal error.
func TestTheHandleTagHoldsTheHandleRule(t *testing.T) {
	t.Parallel()
	const refused = "handle can only use lowercase letters, numbers and underscores"
	cases := []struct{ name, handle, want string }{
		{name: "letters, digits and underscores", handle: "chef_maya_2", want: ""},
		{name: "a space", handle: "chef maya", want: refused},
		{name: "a dash", handle: "chef-maya", want: refused},
		{name: "a capital letter", handle: "Chef_maya", want: refused},
		{name: "an accented letter", handle: "josé", want: refused},
		{name: "an emoji", handle: "chef🍟", want: refused},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := handlers.Validate(struct {
				Handle string `pfield:"handle" validate:"handle"`
			}{c.handle})

			if message(got) != c.want {
				t.Errorf("got %v, want %q", got, c.want)
			}
		})
	}
}

// TestTheEmojiTagTakesOneEmojiAndNothingElse: an avatar shows beside the handle on the board, so
// it must stay a picture and never become a second line of text.
func TestTheEmojiTagTakesOneEmojiAndNothingElse(t *testing.T) {
	t.Parallel()
	const refused = "avatar must be one emoji"
	cases := []struct{ name, avatar, want string }{
		{name: "one emoji", avatar: "🍟", want: ""},
		{name: "a joined emoji", avatar: "👩‍🍳", want: ""},
		{name: "a skin tone", avatar: "👍🏽", want: ""},
		{name: "a flag", avatar: "🇺🇸", want: ""},
		{name: "a subdivision flag", avatar: "🏴󠁧󠁢󠁳󠁣󠁴󠁿", want: ""},
		{name: "emoji presentation", avatar: "❤️", want: ""},
		{name: "the longest, ten code points", avatar: "👩🏻‍❤️‍💋‍👨🏼", want: ""},
		{name: "nothing", avatar: "", want: refused},
		{name: "a letter", avatar: "a", want: refused},
		{name: "an emoji and a letter", avatar: "🍟a", want: refused},
		{name: "a digit", avatar: "7", want: refused},
		{name: "a space between", avatar: "🍟 🍟", want: refused},
		{name: "punctuation", avatar: "!", want: refused},
		{name: "eleven code points", avatar: strings.Repeat("🍟", 11), want: refused},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := handlers.Validate(struct {
				Avatar string `pfield:"avatar" validate:"emoji"`
			}{c.avatar})

			if message(got) != c.want {
				t.Errorf("got %v, want %q", got, c.want)
			}
		})
	}
}

// message is the one violation's text, or "" when there is none.
func message(invalid handlers.FieldErrors) string {
	if len(invalid) == 0 {
		return ""
	}
	return invalid[0].Message
}
