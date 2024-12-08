package repositories

import (
	"context"
	"database/sql"
	"github.com/myrjola/sheerluck/internal/db"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/models"
	"log/slog"
)

type InvestigationRepository struct {
	dbs    *db.DBs
	logger *slog.Logger
}

func NewInvestigationRepository(dbs *db.DBs, logger *slog.Logger) *InvestigationRepository {
	return &InvestigationRepository{
		dbs:    dbs,
		logger: logger.With("source", "InvestigationRepository"),
	}
}

func (r *InvestigationRepository) Get(
	ctx context.Context,
	investigationTargetID string,
	userID []byte,
) (*models.Investigation, error) {
	var (
		investigationTarget models.InvestigationTarget
		completions         []models.Completion
		err                 error
		rows                *sql.Rows
	)

	stmt := `SELECT id, name, short_name, type, image_path FROM investigation_targets WHERE id = ?`
	if err = r.dbs.ReadDB.QueryRowContext(ctx, stmt, investigationTargetID).Scan(
		&investigationTarget.ID,
		&investigationTarget.Name,
		&investigationTarget.ShortName,
		&investigationTarget.Type,
		&investigationTarget.ImagePath,
	); err != nil {
		return nil, errors.Wrap(err, "read investigation target")
	}

	stmt = `SELECT id, "order", question, answer
	FROM completions
	WHERE user_id = ? AND investigation_target_id = ?
	ORDER BY "order"`
	if rows, err = r.dbs.ReadDB.QueryContext(ctx, stmt, userID, investigationTargetID); err != nil {
		return nil, errors.Wrap(err, "query completions")
	}
	defer func() {
		if err = rows.Close(); err != nil {
			err = errors.Wrap(err, "close rows")
			r.logger.Error("could not close rows", errors.SlogError(err))
		}
	}()
	for rows.Next() {
		var (
			completion models.Completion
		)
		if err = rows.Scan(&completion.ID, &completion.Order, &completion.Question, &completion.Answer); err != nil {
			return nil, errors.Wrap(err, "scan completion")
		}
		completions = append(completions, completion)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}

	investigation := models.Investigation{
		Target:      investigationTarget,
		Completions: completions,
	}

	return &investigation, nil
}

func (r *InvestigationRepository) FinishCompletion(
	ctx context.Context,
	investigationTargetID string,
	userID []byte,
	question string,
	answer string,
) error {
	stmt := `WITH new_order AS (SELECT COALESCE(MAX("order") + 1, 0) AS new_order
                   FROM completions
                   WHERE user_id = @user_id
                     AND investigation_target_id = @investigation_target_id)
INSERT
INTO completions (user_id, investigation_target_id, question, answer, "order")
VALUES (@user_id, @investigation_target_id, @question, @answer, (SELECT new_order FROM new_order));`
	params := []any{
		sql.Named("user_id", userID),
		sql.Named("investigation_target_id", investigationTargetID),
		sql.Named("question", question),
		sql.Named("answer", answer),
	}
	if _, err := r.dbs.ReadWriteDB.ExecContext(ctx, stmt, params...); err != nil {
		return errors.Wrap(err, "insert completion")
	}
	return nil
}
