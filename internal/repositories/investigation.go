package repositories

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/myrjola/sheerluck/internal/models"
	"log/slog"
)

type InvestigationRepository struct {
	readWriteDB *sqlx.DB
	readDB      *sqlx.DB
	logger      *slog.Logger
}

func NewInvestigationRepository(readWriteDB *sqlx.DB, readDB *sqlx.DB, logger *slog.Logger) *InvestigationRepository {
	return &InvestigationRepository{
		readWriteDB: readWriteDB,
		readDB:      readDB,
		logger:      logger.With("source", "InvestigationRepository"),
	}
}

func (r *InvestigationRepository) Get(ctx context.Context, investigationTargetID string, userID []byte) (*models.Investigation, error) {
	var (
		investigationTarget models.InvestigationTarget
		completions         []models.Completion
		err                 error
		rows                *sqlx.Rows
	)

	stmt := `SELECT "id", "name", "short_name", "type" FROM "investigation_targets" WHERE "id" = ?`
	if err = r.readDB.GetContext(ctx, &investigationTarget, stmt, investigationTargetID); err != nil {
		return nil, err
	}

	// scan completions for user
	stmt = `SELECT "id", "order", "question", "answer"
FROM "completions"
WHERE "user_id" = ? AND "investigation_target_id" = ?
ORDER BY "order" ASC`
	if rows, err = r.readDB.QueryxContext(ctx, stmt, userID, investigationTargetID); err != nil {
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
			completion models.Completion
		)
		if err = rows.Scan(&completion.ID, &completion.Order, &completion.Question, &completion.Answer); err != nil {
			return nil, err
		}
		completions = append(completions, completion)
	}

	investigation := models.Investigation{
		Target:      investigationTarget,
		Completions: completions,
	}

	return &investigation, nil
}

func (r *InvestigationRepository) FinishCompletion(ctx context.Context, investigationTargetID string, userID []byte, question string, answer string) (*models.Completion, error) {
	var (
		err        error
		completion models.Completion
		tx         *sqlx.Tx
		id         int64
		result     sql.Result
	)
	stmt := `WITH "new_order" AS (SELECT COALESCE(MAX("order") + 1, 0) AS "new_order"
                   FROM "completions"
                   WHERE "user_id" = ?
                     AND "investigation_target_id" = ?)
INSERT
INTO "completions" ("user_id", "investigation_target_id", "order", "question", "answer")
VALUES (?, ?, (SELECT "new_order" FROM "new_order"), ?, ?);`
	if tx, err = r.readWriteDB.BeginTxx(ctx, nil); err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if result, err = tx.ExecContext(ctx, stmt, userID, investigationTargetID, userID, investigationTargetID, question, answer); err != nil {
		return nil, err
	}
	if id, err = result.LastInsertId(); err != nil {
		return nil, err
	}
	stmt = `SELECT "id", "order", "question", "answer" FROM "completions" WHERE "id" = ?`
	if err = tx.GetContext(ctx, &completion, stmt, id); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &completion, nil
}
