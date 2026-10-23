// Author: LAPTOP-BPMQJP10\andyk
package handlers_test

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// TestComboSentinelsKeepTheirMappingsAndHideWrappedDetails pins each newly added error boundary.
func TestComboSentinelsKeepTheirMappingsAndHideWrappedDetails(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		sentinel error
		code     connect.Code
	}{
		{"wrong code", repository.ErrComboCodeMissing, connect.CodeNotFound},
		{"expired", repository.ErrComboExpired, connect.CodeFailedPrecondition},
		{"ended", repository.ErrComboEnded, connect.CodeFailedPrecondition},
		{"not scanned", repository.ErrComboNotReady, connect.CodeFailedPrecondition},
		{"full", repository.ErrComboFull, connect.CodeResourceExhausted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := handlers.ConnectErr(discard(), "join combo", fmt.Errorf("account=7 combo=3: %w", c.sentinel))
			comboError(t, err, c.code, c.sentinel.Error())
		})
	}
}
