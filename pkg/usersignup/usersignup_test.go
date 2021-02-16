package usersignup

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransformUsername(t *testing.T) {
	assertName(t, "some", "some@email.com")
	assertName(t, "so-me", "so-me@email.com")
	assertName(t, "at-email-com", "@email.com")
	assertName(t, "at-crt", "@")
	assertName(t, "some", "some")
	assertName(t, "so-me", "so-me")
	assertName(t, "so-me", "so-----me")
	assertName(t, "so-me", "so_me")
	assertName(t, "so-me", "so me")
	assertName(t, "so-me", "so me@email.com")
	assertName(t, "so-me", "so.me")
	assertName(t, "so-me", "so?me")
	assertName(t, "so-me", "so:me")
	assertName(t, "so-me", "so:#$%!$%^&me")
	assertName(t, "crt-crt", ":#$%!$%^&")
	assertName(t, "some1", "some1")
	assertName(t, "so1me1", "so1me1")
	assertName(t, "crt-me", "-me")
	assertName(t, "crt-me", "_me")
	assertName(t, "me-crt", "me-")
	assertName(t, "me-crt", "me_")
	assertName(t, "crt-me-crt", "_me_")
	assertName(t, "crt-me-crt", "-me-")
	assertName(t, "crt-12345", "12345")
}

var dnsRegExp = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

func assertName(t *testing.T, expected, username string) {
	assert.Regexp(t, dnsRegExp, TransformUsername(username))
	assert.Equal(t, expected, TransformUsername(username))
}
