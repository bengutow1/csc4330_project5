// Author: Caden Lund
package account

import "strings"

// handleInput holds a normalized handle to the handle check on app.account: keep the two in
// step. CreateAccount and CheckHandle both validate through it, so the rule has one copy.
type handleInput struct {
	Handle string `pfield:"handle" validate:"required,min=3,max=20,handle"`
}

// avatarInput holds an avatar to the avatar check on app.account: keep the two in step.
type avatarInput struct {
	Avatar string `pfield:"avatar" validate:"required,emoji"`
}

// normalizeHandle trims and lowercases a handle, before the lookup and before the insert alike,
// so "Chef_Maya " and "chef_maya" are one handle.
func normalizeHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(handle))
}

// normalizeAvatar trims the spaces a keyboard leaves around an emoji.
func normalizeAvatar(avatar string) string {
	return strings.TrimSpace(avatar)
}
