package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func (s *Postgres) MigrateAll(ctx context.Context) error {
	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		raw, err := migrationFiles.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err = s.Pool.Exec(ctx, string(raw)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
