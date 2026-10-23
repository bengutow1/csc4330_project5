// Author: Caden Lund
package profile

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestACursorReadsBackOnlyAsItWasWritten: the next_cursor a page gives out reads back to the same
// key, and anything encodeCursor could not have written is refused rather than read as a page.
func TestACursorReadsBackOnlyAsItWasWritten(t *testing.T) {
	t.Parallel()
	key := repository.PlateKey{At: time.Date(2026, 10, 21, 17, 0, 0, 123456000, time.UTC), PlateID: 42}

	back, ok := decodeCursor(encodeCursor(key))

	if !ok || !back.At.Equal(key.At) || back.PlateID != key.PlateID {
		t.Errorf("round trip = %+v (ok %v), want %+v", back, ok, key)
	}
	for _, raw := range []string{"", "1792602000000000000", "1792602000000000000:", ":42", "01792602000000000000:42",
		"1792602000000000000:042", "1792602000000000000:0", "-5:42", "1792602000000000000:42:7", "now:42"} {
		_, ok := decodeCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)))
		if ok {
			t.Errorf("cursor of %q read as a page, want it refused", raw)
		}
	}
	_, ok = decodeCursor("not base64!")
	if ok {
		t.Error("a cursor that is not base64 read as a page, want it refused")
	}
}
