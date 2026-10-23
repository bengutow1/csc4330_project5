// Author: Caden Lund
package board

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"

	boardv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/board/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// defaultLimit and maxLimit size a page of rows: the podium and a screen of the list, and no
// more than anyone scrolls in one go.
const (
	defaultLimit = 20
	maxLimit     = 50
)

// handleShape is app.account's handle check, which a cursor's handle must pass too.
var handleShape = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)

// getBoardInput is a GetBoard request once it is known good.
type getBoardInput struct {
	board boardv1.Board
	rng   boardv1.Range
	after *repository.BoardKey // nil on the first page
	limit int
	// crewID ranks only that crew's members; 0 is the whole campus.
	crewID int64
}

// boardTags holds the fields a validator tag can judge on its own.
type boardTags struct {
	Limit  int32 `pfield:"limit" validate:"gte=0"`
	CrewID int64 `pfield:"crew_id" validate:"gte=0"`
}

// buildGetBoard validates a GetBoard request, all of it, so the client hears every problem at
// once.
func buildGetBoard(msg *boardv1.GetBoardRequest) (getBoardInput, handlers.FieldErrors) {
	in := getBoardInput{board: msg.Board, rng: msg.Range, limit: defaultLimit, crewID: msg.CrewId}
	invalid := handlers.Validate(boardTags{Limit: msg.Limit, CrewID: msg.CrewId})
	refuse := func(field, message string) {
		invalid = append(invalid, handlers.FieldError{Field: field, Message: message})
	}

	switch msg.Board {
	case boardv1.Board_BOARD_CHEFS, boardv1.Board_BOARD_HALL_OF_FRIES:
	case boardv1.Board_BOARD_UNSPECIFIED:
		refuse("board", "board must be chefs or hall_of_fries")
	default:
		// A value newer than this server.
		refuse("board", "board must be chefs or hall_of_fries")
	}
	switch msg.Range {
	case boardv1.Range_RANGE_THIS_WEEK, boardv1.Range_RANGE_ALL_TIME:
	case boardv1.Range_RANGE_UNSPECIFIED:
		refuse("range", "range must be this_week or all_time")
	default:
		refuse("range", "range must be this_week or all_time")
	}
	if msg.Cursor != "" {
		after, ok := decodeCursor(msg.Cursor)
		if !ok {
			refuse("cursor", "cursor is not one this server gave out")
		}
		in.after = &after
	}
	if msg.Limit > 0 {
		in.limit = min(int(msg.Limit), maxLimit)
	}

	if invalid != nil {
		return getBoardInput{}, invalid
	}
	return in, nil
}

// encodeCursor writes the last row's key as the next_cursor the client sends back.
func encodeCursor(k repository.BoardKey) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(k.Value, 10) + ":" + k.Handle))
}

// decodeCursor reads a cursor back into its key. Anything encodeCursor could not have written is
// refused, rather than read as "start over", which would loop an infinite scroll.
func decodeCursor(cursor string) (repository.BoardKey, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return repository.BoardKey{}, false
	}
	value, handle, found := strings.Cut(string(raw), ":")
	if !found || !handleShape.MatchString(handle) {
		return repository.BoardKey{}, false
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil || v < 1 || strconv.FormatInt(v, 10) != value {
		return repository.BoardKey{}, false
	}
	return repository.BoardKey{Value: v, Handle: handle}, true
}
