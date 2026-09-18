package config

import (
	"encoding/base64"
	"strings"
)

// defaultRedirectPath is where a user lands after login when they didn't ask
// for anywhere in particular.
const defaultRedirectPath = "/dashboard"

// stateNonce is the CSRF part of the oauth state; the post-login path is
// appended to it so it survives the round trip through github.
const stateNonce = "randomstate"

// SafeRedirectPath keeps only paths that stay inside the frontend. Anything
// else - an absolute url, a protocol-relative "//evil.com" - is thrown away so
// login can't be used as an open redirect.
func SafeRedirectPath(path string) string {
	if path == "" || !strings.HasPrefix(path, "/") {
		return defaultRedirectPath
	}
	if strings.HasPrefix(path, "//") || strings.HasPrefix(path, "/\\") {
		return defaultRedirectPath
	}
	return path
}

// EncodeState packs the post-login path into the oauth state parameter.
func EncodeState(path string) string {
	return stateNonce + ":" + base64.RawURLEncoding.EncodeToString([]byte(path))
}

// DecodeState verifies the state github handed back and returns the path to
// send the user to.
func DecodeState(state string) (string, bool) {
	nonce, encoded, found := strings.Cut(state, ":")
	if !found || nonce != stateNonce {
		return "", false
	}

	path, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	return SafeRedirectPath(string(path)), true
}
