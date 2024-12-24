package random

import (
	"crypto/rand"
	"github.com/myrjola/sheerluck/internal/errors"
	"math/big"
)

var allowedLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func Letters(n uint) (string, error) {
	letters := make([]rune, n)
	for i := range letters {
		letterIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", errors.Wrap(err, "generate random integer")
		}
		letters[i] = allowedLetters[letterIndex.Int64()]
	}
	return string(letters), nil
}
