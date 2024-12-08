package webauthnhandler

import (
	"context"
	"encoding/json"
	"github.com/alexedwards/scs/v2"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/myrjola/sheerluck/internal/db"
	"github.com/myrjola/sheerluck/internal/errors"
	"log/slog"
	"net/http"
)

type WebAuthnHandler struct {
	logger         *slog.Logger
	webAuthn       *webauthn.WebAuthn
	sessionManager *scs.SessionManager
	dbs            *db.DBs
}

func New(fqdn string, rpOrigins []string, logger *slog.Logger, sessionManager *scs.SessionManager, dbs *db.DBs) (*WebAuthnHandler, error) {
	var err error

	var webauthnConfig = &webauthn.Config{
		RPDisplayName: "Sheerluck",
		RPID:          fqdn,
		RPOrigins:     rpOrigins,
	}

	var webAuthn *webauthn.WebAuthn
	if webAuthn, err = webauthn.New(webauthnConfig); err != nil {
		return nil, errors.Wrap(err, "new webauthn")
	}

	return &WebAuthnHandler{
		logger:         logger,
		webAuthn:       webAuthn,
		sessionManager: sessionManager,
		dbs:            dbs,
	}, nil
}

func (h *WebAuthnHandler) BeginRegistration(ctx context.Context) ([]byte, error) {
	var (
		user webauthn.User
		err  error
	)
	if user, err = newRandomUser(); err != nil {
		return nil, errors.Wrap(err, "new user")
	}

	authSelect := protocol.AuthenticatorSelection{
		RequireResidentKey: protocol.ResidentKeyNotRequired(),
		UserVerification:   protocol.VerificationDiscouraged,
	}

	opts, session, err := h.webAuthn.BeginRegistration(
		user,
		webauthn.WithAuthenticatorSelection(authSelect),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired))
	if err != nil {
		return nil, errors.Wrap(err, "begin registration")
	}

	h.sessionManager.Put(ctx, string(webAuthnSessionKey), *session)
	if err = h.upsertUser(ctx, user); err != nil {
		return nil, errors.Wrap(err, "upsert user")
	}

	var out []byte
	if out, err = json.Marshal(opts); err != nil {
		return nil, errors.Wrap(err, "JSON encode")
	}
	return out, nil
}

func (h *WebAuthnHandler) parseWebAuthnSession(ctx context.Context) (webauthn.SessionData, error) {
	var (
		session webauthn.SessionData
		ok      bool
		err     error
	)
	if session, ok = h.sessionManager.Get(ctx, string(webAuthnSessionKey)).(webauthn.SessionData); !ok {
		err = errors.New("could not parse webauthn.SessionData")
	}
	return session, err
}

func (h *WebAuthnHandler) FinishRegistration(r *http.Request) error {
	var (
		err     error
		session webauthn.SessionData
		ctx     = r.Context()
	)

	if session, err = h.parseWebAuthnSession(ctx); err != nil {
		return errors.Wrap(err, "parse webauthn session")
	}

	var user webauthn.User
	if user, err = h.getUser(ctx, session.UserID); err != nil {
		return errors.Wrap(err, "get user")
	}

	var credential *webauthn.Credential
	if credential, err = h.webAuthn.FinishRegistration(user, session, r); err != nil {
		return errors.Wrap(err, "finish webauthn registration")
	}

	if err = h.upsertCredential(ctx, user.WebAuthnID(), credential); err != nil {
		return errors.Wrap(err, "upsert webauthn credential")
	}

	// Log in the newly registered user
	if err = h.sessionManager.RenewToken(r.Context()); err != nil {
		return errors.Wrap(err, "renew session token")
	}
	h.sessionManager.Put(r.Context(), string(userIDSessionKey), user.WebAuthnID())

	return nil
}

func (h *WebAuthnHandler) BeginLogin(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	options, session, err := h.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, errors.Wrap(err, "begin discoverable webauthn login")
	}

	h.sessionManager.Put(r.Context(), string(webAuthnSessionKey), *session)

	w.Header().Set("Content-Type", "application/json")
	var out []byte
	if out, err = json.Marshal(options); err != nil {
		return nil, errors.Wrap(err, "json marshal webauthn options")
	}
	return out, nil
}

func (h *WebAuthnHandler) findUserHandler(ctx context.Context) webauthn.DiscoverableUserHandler {
	return func(_, userID []byte) (webauthn.User, error) {
		return h.getUser(ctx, userID)
	}
}

func (h *WebAuthnHandler) FinishLogin(r *http.Request) error {
	var (
		session webauthn.SessionData
		err     error
		user    webauthn.User
		ctx     = r.Context()
	)
	if session, err = h.parseWebAuthnSession(ctx); err != nil {
		return errors.Wrap(err, "parse webauthn session")
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponse(r)
	if err != nil {
		return errors.Wrap(err, "parse credential request response")
	}
	user, credential, err := h.webAuthn.ValidatePasskeyLogin(h.findUserHandler(ctx), session, parsedResponse)
	if err != nil {
		return errors.Wrap(err, "validate PassKey login")
	}

	if err = h.upsertCredential(ctx, user.WebAuthnID(), credential); err != nil {
		return errors.Wrap(err, "upsert webauthn credential")
	}

	// Set userID in session
	if err = h.sessionManager.RenewToken(r.Context()); err != nil {
		return errors.Wrap(err, "renew session token")
	}
	h.sessionManager.Put(r.Context(), string(userIDSessionKey), user.WebAuthnID())

	return nil
}

func (h *WebAuthnHandler) Logout(ctx context.Context) error {
	if err := h.sessionManager.RenewToken(ctx); err != nil {
		return errors.Wrap(err, "renew session token")
	}
	h.sessionManager.Remove(ctx, string(userIDSessionKey))
	return nil
}
