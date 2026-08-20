package gophprofile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type (
	// PgMigrator handles the execution of SQL migration files.
	PgMigrator struct {
		pg *sql.DB
	}

	// MigratorInterface defines the contract for database migration operations.
	MigratorInterface interface {
		Apply(ctx context.Context, dir string, files []string) error
	}
)

// NewPgMigrator creates a new instance of PgMigrator.
func NewPgMigrator(psql *sql.DB) MigratorInterface {
	return &PgMigrator{pg: psql}
}

// Apply reads SQL files from the specified directory within a restricted root
// and executes them against the database. It safely ignores missing files.
func (m *PgMigrator) Apply(ctx context.Context, dir string, files []string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("opening root directory %s: %w", dir, err)
	}
	defer root.Close()

	for _, file := range files {
		baseName := filepath.Base(file)
		fileHandle, err := root.Open(baseName)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("opening migration file %s: %w", baseName, err)
		}
		defer fileHandle.Close()

		content, err := io.ReadAll(fileHandle)
		if err != nil {
			return fmt.Errorf("reading migration file %s: %w", baseName, err)
		}

		_, err = m.pg.ExecContext(ctx, string(content))
		if err != nil {
			return fmt.Errorf("executing migration %s: %w", baseName, err)
		}
	}

	return nil
}
