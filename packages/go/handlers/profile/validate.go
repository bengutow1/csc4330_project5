// Author: Caden Lund
package profile

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	profilev1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/profile/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// defaultLimit and maxLimit size a page of plates: four rows of the grid, and no more than a phone
// shows in two screens. Each plate carries its photo, so a page stays small.
const (
	defaultLimit = 12
	maxLimit     = 24
)

// listPlatesInput is a ListPlates request once it is known good.
type listPlatesInput struct {
	after *repository.PlateKey // nil on the first page
	limit int
}

// limitTag holds the limit to its bound.
type limitTag struct {
	Limit int32 `pfield:"limit" validate:"gte=0"`
}

// buildListPlates validates a ListPlates request, all of it, so the client hears every problem at
// once.
func buildListPlates(msg *profilev1.ListPlatesRequest) (listPlatesInput, handlers.FieldErrors) {
	in := listPlatesInput{limit: defaultLimit}
	invalid := handlers.Validate(limitTag{Limit: msg.Limit})
	if msg.Cursor != "" {
		after, ok := decodeCursor(msg.Cursor)
		if !ok {
			invalid = append(invalid, handlers.FieldError{Field: "cursor", Message: "cursor is not one this server gave out"})
		}
		in.after = &after
	}
	if msg.Limit > 0 {
		in.limit = min(int(msg.Limit), maxLimit)
	}
	if invalid != nil {
		return listPlatesInput{}, invalid
	}
	return in, nil
}

// encodeCursor writes the last plate's key as the next_cursor the client sends back.
func encodeCursor(k repository.PlateKey) string {
	raw := strconv.FormatInt(k.At.UnixNano(), 10) + ":" + strconv.FormatInt(k.PlateID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor reads a cursor back into its key. Anything encodeCursor could not have written is
// refused, rather than read as "start over", which would loop an infinite scroll.
func decodeCursor(cursor string) (repository.PlateKey, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return repository.PlateKey{}, false
	}
	at, id, found := strings.Cut(string(raw), ":")
	if !found {
		return repository.PlateKey{}, false
	}
	nanos, ok := canonical(at, 0)
	if !ok {
		return repository.PlateKey{}, false
	}
	plateID, ok := canonical(id, 1)
	if !ok {
		return repository.PlateKey{}, false
	}
	return repository.PlateKey{At: time.Unix(0, nanos), PlateID: plateID}, true
}

// canonical reads s as a decimal of at least least, written the one way encodeCursor writes it.
func canonical(s string, least int64) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < least || strconv.FormatInt(n, 10) != s {
		return 0, false
	}
	return n, true
}
