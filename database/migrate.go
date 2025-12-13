package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
)

func MigrateUp(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}

	log.Println("Migration UP completed")
	return nil
}

// rollback
func MigrateDown(db *sql.DB, migrationsPath string, steps int) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}

	log.Printf("Migration DOWN %d steps completed\n", steps)
	return nil
}

func GetMigrationVersion(db *sql.DB, migrationsPath string) (uint, bool, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return 0, false, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return 0, false, err
	}

	version, dirty, err := m.Version()
	return version, dirty, err
}

// Example: Safe transaction with context timeout
func SafeTransaction(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// ensure rollback if error
	defer func() {
		if err != nil {
			tx.Rollback()
			log.Println("Transaction rolled back")
		}
	}()

	// execute queries
	_, err = tx.ExecContext(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", "John", "john@example.com")
	if err != nil {
		return err
	}

	// commit transaction
	if err = tx.Commit(); err != nil {
		return err
	}

	log.Println("Transaction committed")
	return nil
}

// Example: Prepared statements (Best Practice - prevent SQL injection)
func GetUserByEmail(db *sql.DB, email string) error {
	stmt, err := db.Prepare("select id, name, email from users where email = $1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	var id int
	var name, userEmail string
	err = stmt.QueryRow(email).Scan(&id, &name, &userEmail)
	if err != nil {
		return nil
	}

	log.Printf("Found user: ID=%d, Name=%s, Email=%s\n", id, name, userEmail)
	return nil
}

// Example: Log slow queries
type LoggingDB struct {
	*sql.DB
}

func (db *LoggingDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.DB.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		log.Printf("SLOW QUERY (%v): %s", duration, query)
	}

	return rows, err
}

// Example: check schema version before running application
func CheckSchemaVersion(db *sql.DB, expectedVersion uint) error {
	var version uint
	err := db.QueryRow("SELECT version FROM schema_migrations LIMIT 1").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}
	if version != expectedVersion {
		return fmt.Errorf("schema version mismatch: got %d, expected %d",
			version, expectedVersion)
	}

	return nil
}
