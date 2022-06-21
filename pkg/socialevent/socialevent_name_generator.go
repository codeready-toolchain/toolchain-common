package socialevent

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// NewName SocialEvent names (a.k.a, activation codes) are generated in a standard format,
// 4 blocks of 4 case insensitive, alphanumeric characters separated by a minus sign delimiter, like so:
// XXXX-XXXX-XXXX-XXXX
// In order to minimise entry errors, a limited character set will be used with visually ambiguous characters excluded:
// Letters:		abcdefghjklmnpqrstuvwxyz
// Figures:		23456789
// This will provide 32^16 possible activation codes, which is expected to be sufficient
// to counter brute force attacks for the typical duration of most events.
func NewName() string {
	chars := []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 'u', 'v', 'w', 'x', 'y', 'z', '2', '3', '4', '5', '6', '7', '8', '9'}
	code := &strings.Builder{}
	for i := 0; i < 4; i++ {
		if code.Len() > 0 { // insert a `-` separator between each group of 4 characters
			code.WriteRune('-')
		}
		for j := 0; j < 4; j++ {
			p, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			code.WriteRune(chars[p.Int64()])
		}
	}
	return code.String()
}
