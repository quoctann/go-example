package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// JSONSeeder - Generic seeder load từ JSON
type JSONSeeder struct {
	TableName  string
	FilePath   string
	Columns    []string
	OnConflict string // DO NOTHING hoặc DO UPDATE
}

func (s *JSONSeeder) Seed(db *sql.DB) error {
	log.Printf("Seeding %s from %s...\n", s.TableName, s.FilePath)

	// Đọc file JSON
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON thành slice of maps
	var records []map[string]interface{}
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(records) == 0 {
		log.Printf("No records found in %s\n", s.FilePath)
		return nil
	}

	// Build INSERT query dynamically
	query := s.buildInsertQuery()

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert từng record
	for _, record := range records {
		values := make([]interface{}, len(s.Columns))
		for i, col := range s.Columns {
			values[i] = record[col]
		}

		if _, err := stmt.Exec(values...); err != nil {
			return fmt.Errorf("failed to insert record: %w", err)
		}
	}

	log.Printf("Seeded %d records into %s\n", len(records), s.TableName)
	return nil
}

func (s *JSONSeeder) buildInsertQuery() string {
	placeholders := ""
	columns := ""

	for i, col := range s.Columns {
		if i > 0 {
			columns += ", "
			placeholders += ", "
		}
		columns += col
		placeholders += fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) %s",
		s.TableName,
		columns,
		placeholders,
		s.OnConflict,
	)

	return query
}

func (s *JSONSeeder) Truncate(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", s.TableName))
	return err
}

// ==================== USAGE ====================

func SeedFromJSON(db *sql.DB) error {
	seeders := []*JSONSeeder{
		{
			TableName:  "users",
			FilePath:   "seeds/users.json",
			Columns:    []string{"name", "email", "password_hash", "status"},
			OnConflict: "ON CONFLICT (email) DO NOTHING",
		},
		{
			TableName:  "categories",
			FilePath:   "seeds/categories.json",
			Columns:    []string{"name", "slug", "description", "is_active"},
			OnConflict: "ON CONFLICT (slug) DO NOTHING",
		},
		{
			TableName:  "settings",
			FilePath:   "seeds/settings.json",
			Columns:    []string{"key", "value", "type", "description"},
			OnConflict: "ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value",
		},
	}

	for _, seeder := range seeders {
		if err := seeder.Seed(db); err != nil {
			return err
		}
	}

	return nil
}

// Environment-specific seeds
// seeds/development/users.json - Nhiều fake data
// seeds/production/users.json  - Chỉ admin users
// seeds/staging/users.json     - Subset của production

func GetSeedPath(env string) string {
	return filepath.Join("seeds", env)
}
