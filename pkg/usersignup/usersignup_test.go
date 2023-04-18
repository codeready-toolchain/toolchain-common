package usersignup

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	assertName(t, "thisisabout19charac", "thisisabout19characters@email.com")
	assertName(t, "isexactly19charactr", "isexactly19charactr@email.com")
	assertName(t, "isexactly19charactr", "isexactly19charactr")
	assertName(t, "isexactly21characte", "isexactly21characte-r")
	assertName(t, "isexactly20charactr", "isexactly20charactr-")
	assertName(t, "thisis19characters", "thisis19characters-")
	assertName(t, "thisis20characters1", "thisis20characters1-")
}

var dnsRegExp = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

func assertName(t *testing.T, expected, username string) {
	assert.Regexp(t, dnsRegExp, TransformUsername(username))
	assert.Equal(t, expected, TransformUsername(username))
}
