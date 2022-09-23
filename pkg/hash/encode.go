package hash

import (
	"crypto/md5" // nolint:gosec
	"encoding/hex"
)

func EncodeString(value string) string {
	return Encode([]byte(value))
}

func Encode(value []byte) string {
	md5hash := md5.New() // nolint:gosec
	// Ignore the error, as this implementation cannot return one
	_, _ = md5hash.Write(value)
	hash := hex.EncodeToString(md5hash.Sum(nil))
	return hash
}
