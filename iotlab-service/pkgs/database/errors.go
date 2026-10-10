package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinels returned by repositories instead of driver errors, so the app layer never imports pgx.
var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate key")
)

// uniqueViolation is the SQLSTATE of a unique or primary key violation.
const uniqueViolation = "23505"

// Translate maps pgx.ErrNoRows to ErrNotFound and a unique violation to ErrDuplicate (wrapping the original). Any other error is
// returned unchanged.
func Translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
		return errors.Join(ErrDuplicate, err)
	}
	return err
}
