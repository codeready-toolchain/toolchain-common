package usersignup

import (
	"fmt"
	validation "k8s.io/apimachinery/pkg/util/validation"
	"regexp"
	"strings"
)

var (
	specialCharRegexp = regexp.MustCompile("[^A-Za-z0-9]")
	onlyNumbers       = regexp.MustCompile("^[0-9]*$")
	// Maximum Length for compliant username is limited to 20 characters such that the result namespace of the type "<compliantUsername>-<ns_suffix>" is less than 30 characters, to be dns compliant.
	// With the AppStudio tier the longest suffix is "-tenant", which is 7 characters, but with subspaces <ns_suffix> is of type "<generatedsuffix(5chars)>-env" making it the longer suffix of 10 chars.
	// Thus the maximum length of compliantUsername should be 30-10 = 20 chars.
	MaxLength = 20
)

func TransformUsername(username string, ForbiddenUsernamePrefixes []string, ForbiddenUsernameSuffixes []string) string {

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
	if len(newUsername) > MaxLength {
		newUsername = newUsername[0:MaxLength]
		if len(validation.IsDNS1123Label(newUsername)) != 0 {
			// trim starting or ending hyphen
			newUsername = strings.Trim(newUsername, "-")
		}
	}

	// -4 for "crt-" or "-crt" to be added in following lines. Update this length changing prefix
	maxlengthWithPrefix := MaxLength - 4
	// Check for any forbidden prefixes
	for _, prefix := range ForbiddenUsernamePrefixes {
		if strings.HasPrefix(newUsername, prefix) {
			if len(newUsername) > maxlengthWithPrefix {
				// replace prefix instead of append
				newUsername = "crt-" + newUsername[3:]
			} else {
				newUsername = fmt.Sprintf("%s%s", "crt-", newUsername)
			}
			break
		}
	}

	// Check for any forbidden suffixes
	for _, suffix := range ForbiddenUsernameSuffixes {
		if strings.HasSuffix(newUsername, suffix) {
			if len(newUsername) > maxlengthWithPrefix {
				// replace prefix instead of append
				newUsername = newUsername[:maxlengthWithPrefix] + "-crt"
			} else {
				newUsername = fmt.Sprintf("%s%s", newUsername, "-crt")
			}
			break
		}
	}
	return newUsername
}
