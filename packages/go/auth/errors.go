// Author: Caden Lund
package auth

import "errors"

// The only errors callers may branch on. Each message goes on screen verbatim, and
// handlers.ConnectErr maps each to its code.
var (
	// ErrNotAuthenticated fires when there is no session, or its account is gone.
	ErrNotAuthenticated = errors.New("not signed in")
	// ErrForbidden fires when the caller is signed in but not allowed to do this.
	ErrForbidden = errors.New("not allowed")
)
