package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type txBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// InTx runs fn using queries bound to one database transaction.
//
// Queries is normally backed by *pgxpool.Pool. Keeping this helper in the db
// package avoids leaking the generated Queries.db field just to coordinate
// higher-level invariants such as quota check + project creation.
func (q *Queries) InTx(ctx context.Context, fn func(*Queries) error) error {
	beginner, ok := q.db.(txBeginner)
	if !ok {
		return fmt.Errorf("database handle does not support transactions")
	}

	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(q.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// LockUserQuota serializes quota-changing operations for a user. Call it only
// from inside InTx. PostgreSQL row locks are preferable to an in-process mutex:
// they remain correct if MyPaas is ever run with more than one API process.
func (q *Queries) LockUserQuota(ctx context.Context, userID uuid.UUID) error {
	var lockedID pgtype.UUID
	if err := q.db.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&lockedID); err != nil {
		return fmt.Errorf("lock user quota: %w", err)
	}
	return nil
}
