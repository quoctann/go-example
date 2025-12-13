package database

import (
	"database/sql"
	"fmt"
	"go-example/config"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

/*
	# Install tool
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

	# Create new migration
	migrate create -ext sql -dir database/migrations -seq create_users_table

	# Run all migration
	migrate -path migrations -database "postgresql://localhost/mydb?sslmode=disable" up

	# Rollback 1 migration
	migrate -path migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" down 1

	# Check current version
	migrate -path migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" version

	# Force version (when dirty - some migration failed, force to specific version)
	migrate -path migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" force 3

	# Can use lib:
	Migration: golang-migrate/migrate
	Query Builder: squirrel, goqu
	ORM: GORM, ent
	Database Driver: lib/pq (PostgreSQL), go-sql-driver/mysql
	Connection Pool: Built-in database/sql
*/

func RunExample(skip bool) {
	if skip {
		return
	}
	cfg, err := config.LoadLocalConfig()
	if err != nil {
		log.Fatal(err)
	}

	migrationsPath := "./database/migrations"

	// Parse command line arguments
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "setup":
		// Setup database: migrate + seed
		err = SetupDatabase(*cfg, migrationsPath, true)

	case "migrate":
		// Chỉ chạy migrations
		db, connErr := connectDB(*cfg)
		if connErr != nil {
			log.Fatal(connErr)
		}
		defer db.Close()
		err = runMigrations(db, migrationsPath)

	case "seed":
		// Chỉ seed data
		db, connErr := connectDB(*cfg)
		if connErr != nil {
			log.Fatal(connErr)
		}
		defer db.Close()
		err = seedDatabase(db)

	case "reset":
		// Reset toàn bộ database
		err = ResetDatabase(*cfg, migrationsPath)

	case "status":
		// Kiểm tra status
		err = CheckDatabaseStatus(*cfg, migrationsPath)

	default:
		printUsage()
		return
	}

	if err != nil {
		log.Fatal("Error: ", err)
	}
}

// runMigrations chạy tất cả migrations
func runMigrations(db *sql.DB, migrationsPath string) error {
	log.Println("\nRunning migrations...")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	// Lấy version hiện tại
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return err
	}

	if dirty {
		log.Printf("Database is in dirty state at version %d\n", currentVersion)
		return fmt.Errorf("database is dirty, please fix manually")
	}

	if err == migrate.ErrNilVersion {
		log.Println("   No migrations applied yet")
	} else {
		log.Printf("   Current version: %d\n", currentVersion)
	}

	// Chạy migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	// Lấy version mới
	newVersion, _, _ := m.Version()
	log.Printf("Migrations completed (version: %d)\n", newVersion)

	return nil
}

func seedDatabase(db *sql.DB) error {
	log.Println("\nSeeding database...")

	seeder := NewMasterDataSeeder(db)
	if err := seeder.Run(); err != nil {
		return err
	}

	return nil
}

func connectDB(cfg config.Config) (*sql.DB, error) {
	c := cfg.Database
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// connection pool setting (best practice)
	db.SetMaxOpenConns(25)                 // max 25 connections
	db.SetMaxIdleConns(5)                  // keep 5 idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Connection max existing on 5 mins

	// check connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	log.Println("Database connected successfully")
	return db, nil
}

func SetupDatabase(cfg config.Config, migrationsPath string, shouldSeed bool) error {
	log.Println("Starting Database Setup")

	// 1. Kết nối database
	db, err := connectDB(cfg)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	// 2. Chạy migrations
	if err := runMigrations(db, migrationsPath); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// 3. Seed data nếu cần
	if shouldSeed {
		if err := seedDatabase(db); err != nil {
			return fmt.Errorf("seeding failed: %w", err)
		}
	}

	log.Println("Database Setup Completed Successfully!")
	return nil
}

// ResetDatabase xóa và tạo lại database từ đầu
func ResetDatabase(cfg config.Config, migrationsPath string) error {
	log.Println("RESET DATABASE - This will delete all data!")

	db, err := connectDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	// Rollback tất cả migrations
	log.Println("\nRolling back all migrations...")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("All migrations rolled back")

	// Chạy lại migrations
	if err := runMigrations(db, migrationsPath); err != nil {
		return err
	}

	// Seed data
	if err := seedDatabase(db); err != nil {
		return err
	}

	log.Println("Database Reset Completed!")
	return nil
}

// CheckDatabaseStatus kiểm tra trạng thái database
func CheckDatabaseStatus(cfg config.Config, migrationsPath string) error {
	db, err := connectDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Println("\nDatabase Status:")

	// Migration version
	driver, _ := postgres.WithInstance(db, &postgres.Config{})
	m, _ := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)

	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		log.Println("   Migration version: Not initialized")
	} else {
		log.Printf("   Migration version: %d (dirty: %v)\n", version, dirty)
	}

	// Đếm records trong các tables
	tables := []string{"users", "categories", "products", "settings", "countries", "orders"}
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := db.QueryRow(query).Scan(&count)
		if err != nil {
			log.Printf("   %s: table not exists or error\n", table)
		} else {
			log.Printf("   %s: %d records\n", table, count)
		}
	}

	return nil
}

func printUsage() {
	fmt.Println("Usage: go run setup.go [command]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  setup   - Run migrations and seed data (full setup)")
	fmt.Println("  migrate - Run migrations only")
	fmt.Println("  seed    - Seed data only")
	fmt.Println("  reset   - Reset database (drop all, migrate, seed)")
	fmt.Println("  status  - Check database status")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run setup.go setup   # First time setup")
	fmt.Println("  go run setup.go migrate # Run new migrations")
	fmt.Println("  go run setup.go seed    # Seed master data")
	fmt.Println("  go run setup.go reset   # Reset everything")
	fmt.Println("  go run setup.go status  # Check status")
}
