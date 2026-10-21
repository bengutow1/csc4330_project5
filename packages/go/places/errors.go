// Author: Caden Lund
package places

import "errors"

var (
	// ErrUnavailable fires when the provider fails or runs out of time. Trying again may work.
	ErrUnavailable = errors.New("couldn't look up spots nearby, try again")
)
