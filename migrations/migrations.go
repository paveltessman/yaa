package migrations

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed sql/*.sql
var migrations embed.FS

const MigrationsDir = "sql"

func Migrations() (fs.FS, error) {
	sub, err := fs.Sub(migrations, MigrationsDir)
	if err != nil {
		return nil, fmt.Errorf("migrations: can't root the fs at %s: %w", MigrationsDir, err)
	}
	return sub, nil
}
