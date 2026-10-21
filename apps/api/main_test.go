// Author: Caden Lund
package main

import "testing"

// TestOnlyAnExactLocalDevAppEnvDropsTheSecureCookie: the session cookie is the account's only
// credential, so a deployment that forgot APP_ENV, or misspelled it, must still send it over
// https only.
func TestOnlyAnExactLocalDevAppEnvDropsTheSecureCookie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		appEnv string
		secure bool
	}{
		{name: "unset", appEnv: "", secure: true},
		{name: "production", appEnv: "production", secure: true},
		{name: "misspelled", appEnv: "developmnet", secure: true},
		{name: "abbreviated", appEnv: "dev", secure: true},
		{name: "capitalized", appEnv: "Development", secure: true},
		{name: "padded", appEnv: " local", secure: true},
		{name: "development", appEnv: "development", secure: false},
		{name: "local", appEnv: "local", secure: false},
	}
	for _, c := range cases {
		// main.go sets the cookie's Secure flag to !isLocalDev(APP_ENV).
		secure := !isLocalDev(c.appEnv)
		if secure != c.secure {
			t.Errorf("%s: APP_ENV=%q gives Secure = %v, want %v", c.name, c.appEnv, secure, c.secure)
		}
	}
}
