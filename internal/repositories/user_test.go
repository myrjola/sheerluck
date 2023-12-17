package repositories

import (
	"context"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/myrjola/sheerluck/internal/models"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"testing"
)

func TestUserRepository(t *testing.T) {
	db := newTestDB(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := NewUserRepository(db, logger)

	user, err := models.NewUser()
	if err != nil {
		t.Fatal(err)
	}

	userWithCredentials, err := models.NewUser()
	if err != nil {
		t.Fatal(err)
	}
	userWithCredentials.AddWebAuthnCredential(webauthn.Credential{
		ID:              []byte{1, 2, 3},
		PublicKey:       []byte{4, 5, 6},
		AttestationType: "none",
		Transport:       []protocol.AuthenticatorTransport{protocol.Internal, protocol.Hybrid},
		Flags: webauthn.CredentialFlags{
			UserPresent:    true,
			UserVerified:   false,
			BackupEligible: true,
			BackupState:    false,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:       []byte{7, 8, 9},
			SignCount:    3,
			CloneWarning: false,
			Attachment:   "cross-platform",
		},
	})

	tests := []struct {
		name string
		user *models.User
	}{
		{
			name: "user without credentials",
			user: user,
		},
		{
			name: "user with credentials",
			user: userWithCredentials,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				readUser *models.User
				err      error
				ctx      = context.Background()
			)
			err = repo.Create(ctx, tt.user)
			require.NoError(t, err, "failed to create user")

			readUser, err = repo.Get(tt.user.ID)
			require.NoError(t, err, "failed to read user")
			require.NotNilf(t, readUser, "user not found")
			require.Equal(t, readUser.ID, tt.user.ID)
			require.Equal(t, readUser.DisplayName, tt.user.DisplayName)
			require.Equal(t, readUser.WebAuthnCredentials(), tt.user.WebAuthnCredentials())
		})
	}
}
