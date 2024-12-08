package webauthnhandler

type sessionKey string

const webAuthnSessionKey = sessionKey("webauthn")
const userIDSessionKey = sessionKey("userID")
