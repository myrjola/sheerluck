package repositories

import (
	"github.com/myrjola/sheerluck/internal/models"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"testing"
)

func TestUserRepository(t *testing.T) {
	db := newTestDB(t)

	t.Run("creates a user and reads it back", func(t *testing.T) {
		var (
			user     *models.User
			readUser *models.User
			err      error
		)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		repo := NewUserRepository(db, logger)
		if user, err = models.NewUser(); err != nil {
			t.Fatal(err)
		}

		if err := repo.Create(*user); err != nil {
			t.Fatal(err)
		}

		if readUser, err = repo.Get(user.ID); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, readUser.ID, user.ID)
		assert.Equal(t, readUser.DisplayName, user.DisplayName)
		assert.Empty(t, readUser.WebAuthnCredentials())
	})
}
