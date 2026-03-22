package misc

import (
	"math/rand"
)

// TokenGen generates token string
func TokenGen(tokenLen int64) string {

	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, tokenLen)

	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}

	return string(b)
}
