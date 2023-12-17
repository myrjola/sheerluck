package models

import (
	"crypto/rand"
	"fmt"
	"github.com/go-webauthn/webauthn/webauthn"
	"time"
)

type User struct {
	ID          []byte `db:"id"`
	DisplayName string `db:"display_name"`
	Credentials []webauthn.Credential
}

func NewUser() (*User, error) {
	id := make([]byte, 64)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	user := User{
		DisplayName: fmt.Sprintf("Anonymous user created at %s", time.Now().Format(time.RFC3339)),
		ID:          id,
		Credentials: []webauthn.Credential{},
	}

	return &user, nil
}

// WebAuthnID provides the user handle of the user account. A user handle is an opaque byte sequence with a maximum
// size of 64 bytes, and is not meant to be displayed to the user.
//
// To ensure secure operation, authentication and authorization decisions MUST be made on the basis of this id
// member, not the displayName nor name members. See Section 6.1 of [RFC8266].
//
// It's recommended this value is completely random and uses the entire 64 bytes.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dom-publickeycredentialuserentity-id)
func (u User) WebAuthnID() []byte {
	return u.ID
}

// WebAuthnName provides the name attribute of the user account during registration and is a human-palatable name for the user
// account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party SHOULD let the user
// choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dictdef-publickeycredentialuserentity)
func (u User) WebAuthnName() string {
	return u.DisplayName
}

// WebAuthnDisplayName provides the name attribute of the user account during registration and is a human-palatable
// name for the user account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party
// SHOULD let the user choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialuserentity-displayname)
func (u User) WebAuthnDisplayName() string {
	return u.DisplayName
}

// WebAuthnCredentials provides the list of Credentials owned by the user.
func (u User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// AddWebAuthnCredentials adds Credential to the user.
func (u *User) AddWebAuthnCredential(credential webauthn.Credential) {
	u.Credentials = append(u.Credentials, credential)
}

// WebAuthnIcon is a deprecated option.
// Deprecated: this has been removed from the specification recommendation. Suggest a blank string.
func (u User) WebAuthnIcon() string {
	return ""
}
