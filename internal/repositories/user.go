package repositories

import (
	"context"
	"encoding/json"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jmoiron/sqlx"
	"github.com/myrjola/sheerluck/internal/models"
	"log/slog"
)

type UserRepository struct {
	db     *sqlx.DB
	logger *slog.Logger
}

func NewUserRepository(db *sqlx.DB, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) Get(id []byte) (*models.User, error) {
	var (
		user models.User
		err  error
		rows *sqlx.Rows
	)

	user.Credentials = make([]webauthn.Credential, 0)

	stmt := `SELECT id, display_name FROM users WHERE id = ?`
	if err = r.db.Get(&user, stmt, id); err != nil {
		return nil, err
	}

	// scan credentials
	stmt = `SELECT id,
       public_key,
       attestation_type,
       transport,
       flag_user_present,
       flag_user_verified,
       flag_backup_eligible,
       flag_backup_state,
       authenticator_aaguid,
       authenticator_sign_count,
       authenticator_clone_warning,
       authenticator_attachment
FROM credentials
WHERE user_id = ?`
	if rows, err = r.db.Queryx(stmt, id); err != nil {
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			r.logger.Error("could not close rows", "err", err)
		}
	}()

	for rows.Next() {
		var (
			credential webauthn.Credential
			transport  []byte
		)
		if err = rows.Scan(&credential.ID, &credential.PublicKey, &credential.AttestationType, &transport, &credential.Flags.UserPresent, &credential.Flags.UserVerified, &credential.Flags.BackupEligible, &credential.Flags.BackupState, &credential.Authenticator.AAGUID, &credential.Authenticator.SignCount, &credential.Authenticator.CloneWarning, &credential.Authenticator.Attachment); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(transport, &credential.Transport); err != nil {
			return nil, err
		}
		user.AddWebAuthnCredential(credential)
	}

	return &user, nil
}

func (r *UserRepository) Upsert(ctx context.Context, user *models.User) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {

	}
	defer func() {
		_ = tx.Rollback()
	}()
	stmt := `INSERT INTO users (id, display_name) VALUES (:id, :display_name) ON CONFLICT DO NOTHING`
	_, err = tx.NamedExecContext(ctx, stmt, user)
	if err != nil {
		return err
	}

	// Upsert credentials
	stmt = `INSERT INTO credentials (id,
                         user_id,
                         public_key,
                         attestation_type,
                         transport,
                         flag_user_present,
                         flag_user_verified,
                         flag_backup_eligible,
                         flag_backup_state,
                         authenticator_aaguid,
                         authenticator_sign_count,
                         authenticator_clone_warning,
                         authenticator_attachment)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
        $11, $12, $13)
ON CONFLICT (id) DO UPDATE SET attestation_type            = EXCLUDED.attestation_type,
                               transport                   = EXCLUDED.transport,
                               flag_user_present           = EXCLUDED.flag_user_present,
                               flag_user_verified          = EXCLUDED.flag_user_verified,
                               flag_backup_eligible        = EXCLUDED.flag_backup_eligible,
                               flag_backup_state           = EXCLUDED.flag_backup_state,
                               authenticator_aaguid        = EXCLUDED.authenticator_aaguid,
                               authenticator_sign_count    = EXCLUDED.authenticator_sign_count,
                               authenticator_clone_warning = EXCLUDED.authenticator_clone_warning,
                               authenticator_attachment    = EXCLUDED.authenticator_attachment;
                                 
                                   `
	for _, c := range user.WebAuthnCredentials() {
		encodedTransport, err := json.Marshal(c.Transport)
		if err != nil {
			return err
		}
		_, err = tx.Exec(stmt, c.ID, user.ID, c.PublicKey, c.AttestationType, string(encodedTransport), c.Flags.UserPresent, c.Flags.UserVerified, c.Flags.BackupEligible, c.Flags.BackupState, c.Authenticator.AAGUID, c.Authenticator.SignCount, c.Authenticator.CloneWarning, c.Authenticator.Attachment)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *UserRepository) Exists(id []byte) (bool, error) {
	var (
		err error
		row *sqlx.Row
	)

	stmt := `SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)`
	if row = r.db.QueryRowx(stmt, id); err != nil {
		return false, err
	}

	var exists bool
	if err = row.Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
