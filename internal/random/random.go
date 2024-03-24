package random

import (
	"crypto/rand"
	"math/big"
)

var allowedLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomLetters(n uint) (string, error) {
	letters := make([]rune, n)
	for i := range letters {
		letterIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		letters[i] = allowedLetters[letterIndex.Int64()]
	}
	return string(letters), nil
}
