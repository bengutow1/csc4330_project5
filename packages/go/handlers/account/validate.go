// Author: Caden Lund
package account

import "strings"

// avatarInput holds an avatar to the avatar check on app.account: keep the two in step.
type avatarInput struct {
	Avatar string `pfield:"avatar" validate:"required,emoji"`
}

// normalizeAvatar trims the spaces a keyboard leaves around an emoji.
func normalizeAvatar(avatar string) string {
	return strings.TrimSpace(avatar)
}
