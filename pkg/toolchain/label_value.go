package toolchain

import "strings"

// ToValidValue takes a string and converts it to a compliant DNS-1123 value.
func ToValidValue(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "@", "-at-"), ".", "-")
}
