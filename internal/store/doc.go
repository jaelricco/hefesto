// Package store is the persistence layer: the pgx pool, sqlc-generated
// queries in dbgen, and the few operations that span many queries in one
// transaction (content seeding first among them).
//
// SQL lives in db/queries/*.sql; run `make sqlc` after changing it or the
// schema. Integration tests in this package hit a real Postgres.
package store
