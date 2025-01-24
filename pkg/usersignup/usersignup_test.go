package usersignup

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"

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
	assertName(t, "thisisabout20charact", "thisisabout20characters@email.com")
	assertName(t, "isexactly20character", "isexactly20character@email.com")
	assertName(t, "isexactly20character", "isexactly20character")
	assertName(t, "isexactly21characte", "isexactly21characte-r") // shortened username would've end in hyphen, should be trimmed
	assertName(t, "isexactly20charactr", "isexactly20charactr-")  // but ending in hyphen
	assertName(t, "thisis19characters-c", "thisis19characters-")  // suffix -crt is added before truncating string
	assertName(t, "john-crtadmin-crt", "john-crtadmin")           // forbidden suffix
	assertName(t, "johny-long-crtad-crt", "johny-long-crtadmin-") // forbidden suffix with username exactly of maxLength
	assertName(t, "crt-nshift-test-user", "openshift-test-user")  // forbidden prefix in username, transforms to replace in place
	assertName(t, "crt-kube-test-user", "kube-test-user")         // forbidden prefix username, transforms to prepend crt-
}

func assertName(t *testing.T, expected, username string) {
	assert.Empty(t, validation.IsDNS1123Label(TransformUsername(username, []string{"openshift", "kube", "default", "redhat", "sandbox"}, []string{"admin"})), "username is not a compliant DNS label")
	assert.Equal(t, expected, TransformUsername(username, []string{"openshift", "kube", "default", "redhat", "sandbox"}, []string{"admin"}))
}

func TestEncodeUsername(t *testing.T) {
	assertEncodedUsername(t, "abcde-12345", "abcde-12345")
	assertEncodedUsername(t, "c0177ca4-abcde-12345", "abcde\\*-12345")
	assertEncodedUsername(t, "ca3e1e0f-1234567", "-1234567")
	assertEncodedUsername(t, "e3632025-0123456789abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqr", "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-01234567890123456789")
	assertEncodedUsername(t, "a05a4053-abcxyz", "abc:xyz")
	assertEncodedUsername(t, "ed6bd2b5-abc", "abc---")
	assertEncodedUsername(t, "8fa24710-johnnykubesawcom", "johnny@kubesaw.com")
}

func assertEncodedUsername(t *testing.T, expected, username string) {
	assert.Equal(t, expected, EncodeUserIdentifier(username))
}
