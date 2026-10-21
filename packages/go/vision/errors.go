// Author: Caden Lund
package vision

import "errors"

// The only error callers may branch on. Its message goes on screen verbatim.
var (
	// ErrUnavailable fires when the model fails or runs out of time. Nothing is wrong with what
	// the player sent, so trying again may work.
	ErrUnavailable = errors.New("couldn't judge that photo, try again")
)
