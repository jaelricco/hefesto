package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jaelricco/hefesto/internal/domain/training"
)

// Errors the store returns for conditions a caller is expected to handle.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrEmailTaken    = errors.New("email already registered")
	ErrOrderConflict = errors.New("another live sibling has this order_index")
	ErrDeviceTaken   = errors.New("device id belongs to another account")
)

const (
	pgUniqueViolation = "23505"
	pgCheckViolation  = "23514"
	pgFKViolation     = "23503"
)

// translate maps database errors onto the store's vocabulary. Anything it
// does not recognise is wrapped and returned as-is: an unexpected constraint
// violation is a bug to see in the logs, not a 422 to hide.
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch {
	case pg.Code == pgUniqueViolation && strings.HasSuffix(pg.ConstraintName, "_order_uk"):
		return fmt.Errorf("%w (%s)", ErrOrderConflict, pg.ConstraintName)
	case pg.Code == pgUniqueViolation && pg.ConstraintName == "users_email_key":
		return ErrEmailTaken
	case pg.Code == pgCheckViolation || pg.Code == pgFKViolation:
		return &training.ValidationError{Fields: training.FieldErrors{
			"": fmt.Sprintf("rejected by the database (%s)", pg.ConstraintName),
		}}
	}
	return err
}

func isUnique(err error, constraint string) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == pgUniqueViolation && pg.ConstraintName == constraint
}
