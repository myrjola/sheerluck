package repositories

import (
	"context"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/myrjola/sheerluck/internal/models"
	"log/slog"
)

type UserRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewUserRepository(db *pgxpool.Pool, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) Get(id []byte) (*models.User, error) {
	var user models.User
	stmt := `SELECT 
    u.id,
    u.display_name,
    c.id,
    c.public_key,
    c.attestation_type,
    c.transport,
    c.flag_user_present,
    c.flag_user_verified,
    c.flag_backup_eligible,
    c.flag_backup_state,
    c.authenticator_aaguid,
    c.authenticator_sign_count,
    c.authenticator_clone_warning,
    c.authenticator_attachment
FROM users u
LEFT JOIN credentials c ON u.id = c.user_id
WHERE u.id = $1`
	rows, err := r.db.Query(context.Background(), stmt, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	if err = rows.Scan(&user.ID, &user.DisplayName, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		return nil, err
	}

	for {
		var c webauthn.Credential
		err = rows.Scan(nil, nil, &c.ID, &c.PublicKey, &c.AttestationType, &c.Transport, &c.Flags.UserPresent, &c.Flags.UserVerified, &c.Flags.BackupEligible, &c.Flags.BackupState, &c.Authenticator.AAGUID, &c.Authenticator.SignCount, &c.Authenticator.CloneWarning, &c.Authenticator.Attachment)
		if err != nil {
			return nil, err
		}
		if !rows.Next() {
			break
		}
	}

	return &user, nil
}

func (r *UserRepository) Create(user models.User) error {
	tx, err := r.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {

	}
	defer tx.Rollback(context.Background())
	stmt := `INSERT INTO users (id, display_name) VALUES ($1, $2)`
	_, err = tx.Exec(context.Background(), stmt, user.ID, user.DisplayName)
	if err != nil {
		return err
	}

	// upsert credentials
	stmt = `INSERT INTO credentials (
    id,
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
    authenticator_attachment
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13) ON CONFLICT DO NOTHING`
	for _, c := range user.Credentials {
		_, err = tx.Exec(context.Background(), stmt, c.ID, user.ID, c.PublicKey, c.AttestationType, c.Transport, c.Flags.UserPresent, c.Flags.UserVerified, c.Flags.BackupEligible, c.Flags.BackupState, c.Authenticator.AAGUID, c.Authenticator.SignCount, c.Authenticator.CloneWarning, c.Authenticator.Attachment)
		if err != nil {
			return err
		}
	}

	return tx.Commit(context.Background())
}
