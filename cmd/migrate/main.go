// Command migrate applies or rolls back database migrations using
// golang-migrate, reading .sql files from a migrations directory.
//
// Usage:
//
//	go run ./cmd/migrate -database "<postgres DSN>" -path ./migrations up
//	go run ./cmd/migrate -database "<postgres DSN>" -path ./migrations down 1
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	databaseURL := flag.String("database", "", "PostgreSQL connection URL")
	migrationsPath := flag.String("path", "migrations", "path to migration files")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("migrate: -database is required")
	}

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("migrate: expected a command: up | down [N] | version | force <N>")
	}

	sourceURL := "file://" + *migrationsPath

	m, err := migrate.New(sourceURL, *databaseURL)
	if err != nil {
		log.Fatalf("migrate: failed to initialize: %v", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("migrate: source close error: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("migrate: db close error: %v", dbErr)
		}
	}()

	if err := run(m, args); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("migrate: no change")
			return
		}
		log.Fatalf("migrate: %v", err)
	}

	fmt.Println("migrate: done")
}

func run(m *migrate.Migrate, args []string) error {
	switch args[0] {
	case "up":
		return m.Up()
	case "down":
		steps := 1
		if len(args) > 1 {
			n, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid step count %q: %w", args[1], err)
			}
			steps = n
		}
		return m.Steps(-steps)
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			return err
		}
		fmt.Printf("version=%d dirty=%t\n", version, dirty)
		return nil
	case "force":
		if len(args) < 2 {
			return errors.New("force requires a version argument")
		}
		version, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid version %q: %w", args[1], err)
		}
		return m.Force(version)
	default:
		return fmt.Errorf("unknown command %q: expected up | down [N] | version | force <N>", args[0])
	}
}
