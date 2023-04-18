package usersignup

import (
	"regexp"
	"strings"
)

var (
	specialCharRegexp = regexp.MustCompile("[^A-Za-z0-9]")
	onlyNumbers       = regexp.MustCompile("^[0-9]*$")
	dnsRegexp         = regexp.MustCompile("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$")
	// Maximum Length for compliant username is limited to 19 characters such that the result namespace of the type <compliantUsername>-<ns_suffix> is less than 30 characters, to be dns compliant.
	// With the AppStudio tier the longest suffix is -tenant, which is 7 characters, but with subspaces an additional -env is 4 characters.
	maxLength = 19
)

func TransformUsername(username string) string {
	newUsername := specialCharRegexp.ReplaceAllString(strings.Split(username, "@")[0], "-")
	if len(newUsername) == 0 {
		newUsername = strings.ReplaceAll(username, "@", "at-")
	}
	newUsername = specialCharRegexp.ReplaceAllString(newUsername, "-")

	matched := onlyNumbers.MatchString(newUsername)
	if matched {
		newUsername = "crt-" + newUsername
	}
	for strings.Contains(newUsername, "--") {
		newUsername = strings.ReplaceAll(newUsername, "--", "-")
	}
	if strings.HasPrefix(newUsername, "-") {
		newUsername = "crt" + newUsername
	}
	if strings.HasSuffix(newUsername, "-") {
		newUsername = newUsername + "crt"
	}
	if len(newUsername) > maxLength {
		newUsername = newUsername[0:maxLength]
		if !dnsRegexp.MatchString(newUsername) {
			// trim starting or ending hyphen
			newUsername = strings.Trim(newUsername, "-")
		}
	}
	return newUsername
}
