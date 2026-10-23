// Author: Arthur
package handlers

import "strings"

// HandleInput holds a normalized handle to the handle check on app.account: keep the two in step.
// Every rpc that takes a handle (making an account, checking one, opening a player's card)
// validates through it, so the rule has one copy.
type HandleInput struct {
	Handle string `pfield:"handle" validate:"required,min=3,max=20,handle"`
}

// NormalizeHandle trims and lowercases a handle before any lookup or insert, so "Chef_Maya " and
// "chef_maya" are one handle.
func NormalizeHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(handle))
}
