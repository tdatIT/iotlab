// Package migrations holds the versioned SQL schema migrations applied by golang-migrate. The .sql files are embedded into the
// binary so the service is self-contained — no migration directory needs to ship alongside the image.
//
// File naming follows golang-migrate's convention:
//
//	NNNNNN_<description>.up.sql   — applied on migrate up
//	NNNNNN_<description>.down.sql — applied on rollback
package migrations

import "embed"

// Files holds every embedded .sql migration, passed to database.RunMigrations
// at startup.
//
//go:embed *.sql
var Files embed.FS
