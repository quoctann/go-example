package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Seeder interface {
	Seed(db *sql.DB) error
	Truncate(db *sql.DB) error
}

type MasterDataSeeder struct {
	db      *sql.DB
	seeders []Seeder
}

func NewMasterDataSeeder(db *sql.DB) *MasterDataSeeder {
	return &MasterDataSeeder{
		db: db,
		seeders: []Seeder{
			&UserSeeder{},
			&CategorySeeder{},
			&ProductSeeder{},
			&SettingSeeder{},
		},
	}
}

// Run all seeder
func (s *MasterDataSeeder) Run() error {
	log.Println("Starting database seeding...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			log.Println("Seeding failed, rolled back")
		}
	}()

	// run every seeder
	for _, seeder := range s.seeders {
		if err := seeder.Seed(s.db); err != nil {
			return fmt.Errorf("seeder failed: %w", err)
		}
	}

	// commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Println("Database seeding completed successfully!")
	return nil
}

// Truncate delete all data (for reseed)
func (s *MasterDataSeeder) Truncate() error {
	log.Println("Truncating all tables...")

	// run reverse to prevent foreign key constraint
	for i := len(s.seeders) - 1; i >= 0; i-- {
		if err := s.seeders[i].Truncate(s.db); err != nil {
			return fmt.Errorf("truncate failed: %w", err)
		}
	}

	log.Println("All tables truncated")
	return nil
}

// ==================== USER SEEDER ====================

type User struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

type UserSeeder struct{}

func (s *UserSeeder) Seed(db *sql.DB) error {
	log.Println("Seeding users...")

	// Option 1: Hardcode trong code
	// users := []User{
	// 	{Name: "Admin User", Email: "admin@example.com", Password: "hashed_password_1", Status: "active"},
	// 	{Name: "John Doe", Email: "john@example.com", Password: "hashed_password_2", Status: "active"},
	// 	{Name: "Jane Smith", Email: "jane@example.com", Password: "hashed_password_3", Status: "active"},
	// 	{Name: "Test User", Email: "test@example.com", Password: "hashed_password_4", Status: "inactive"},
	// }

	// Option 2: Load từ JSON file
	users, err := s.loadFromJSON("database/seeds/users.json")
	if err != nil {
		return err
	}

	// Insert data
	query := `
		INSERT INTO users (name, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (email) DO NOTHING
	`

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, user := range users {
		_, err := stmt.Exec(user.Name, user.Email, user.Password, user.Status)
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", user.Email, err)
		}
	}

	log.Printf("Seeded %d users\n", len(users))
	return nil
}

func (s *UserSeeder) Truncate(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE TABLE users CASCADE")
	return err
}

func (s *UserSeeder) loadFromJSON(filepath string) ([]User, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(file, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// ==================== CATEGORY SEEDER ====================

type Category struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
}

type CategorySeeder struct{}

func (s *CategorySeeder) Seed(db *sql.DB) error {
	log.Println("Seeding categories...")

	categories := []Category{
		{Name: "Electronics", Description: "Electronic devices and accessories", Slug: "electronics"},
		{Name: "Clothing", Description: "Fashion and apparel", Slug: "clothing"},
		{Name: "Books", Description: "Books and magazines", Slug: "books"},
		{Name: "Home & Garden", Description: "Home improvement and garden tools", Slug: "home-garden"},
		{Name: "Sports", Description: "Sports equipment and gear", Slug: "sports"},
	}

	query := `
		INSERT INTO categories (name, description, slug, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (slug) DO NOTHING
	`

	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, cat := range categories {
		_, err := stmt.Exec(cat.Name, cat.Description, cat.Slug)
		if err != nil {
			return err
		}
	}

	log.Printf("Seeded %d categories\n", len(categories))
	return nil
}

func (s *CategorySeeder) Truncate(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE TABLE categories CASCADE")
	return err
}

// ==================== PRODUCT SEEDER ====================

type Product struct {
	Name        string  `json:"name"`
	CategoryID  int     `json:"category_id"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Description string  `json:"description"`
}

type ProductSeeder struct{}

func (s *ProductSeeder) Seed(db *sql.DB) error {
	log.Println("Seeding products...")

	// Lấy category IDs trước
	categoryMap, err := s.getCategoryIDs(db)
	if err != nil {
		return err
	}

	products := []Product{
		{Name: "Laptop Dell XPS 13", CategoryID: categoryMap["electronics"], Price: 1299.99, Stock: 50, Description: "High-end laptop"},
		{Name: "iPhone 15 Pro", CategoryID: categoryMap["electronics"], Price: 999.99, Stock: 100, Description: "Latest iPhone"},
		{Name: "T-Shirt Cotton", CategoryID: categoryMap["clothing"], Price: 19.99, Stock: 200, Description: "Comfortable cotton t-shirt"},
		{Name: "Jeans Blue", CategoryID: categoryMap["clothing"], Price: 49.99, Stock: 150, Description: "Classic blue jeans"},
		{Name: "The Go Programming Language", CategoryID: categoryMap["books"], Price: 39.99, Stock: 75, Description: "Book by Alan Donovan"},
	}

	query := `
		INSERT INTO products (name, category_id, price, stock, description, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT DO NOTHING
	`

	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, product := range products {
		_, err := stmt.Exec(product.Name, product.CategoryID, product.Price, product.Stock, product.Description)
		if err != nil {
			return err
		}
	}

	log.Printf("Seeded %d products\n", len(products))
	return nil
}

func (s *ProductSeeder) getCategoryIDs(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query("SELECT id, slug FROM categories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryMap := make(map[string]int)
	for rows.Next() {
		var id int
		var slug string
		if err := rows.Scan(&id, &slug); err != nil {
			return nil, err
		}
		categoryMap[slug] = id
	}

	return categoryMap, nil
}

func (s *ProductSeeder) Truncate(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE TABLE products CASCADE")
	return err
}

// ==================== SETTING SEEDER (Key-Value Config) ====================

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type SettingSeeder struct{}

func (s *SettingSeeder) Seed(db *sql.DB) error {
	log.Println("Seeding settings...")

	settings := []Setting{
		{Key: "site_name", Value: "My E-commerce Store", Type: "string"},
		{Key: "site_email", Value: "admin@store.com", Type: "string"},
		{Key: "tax_rate", Value: "0.1", Type: "float"},
		{Key: "currency", Value: "USD", Type: "string"},
		{Key: "maintenance_mode", Value: "false", Type: "boolean"},
		{Key: "max_upload_size", Value: "5242880", Type: "integer"},
	}

	query := `
		INSERT INTO settings (key, value, type, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`

	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, setting := range settings {
		_, err := stmt.Exec(setting.Key, setting.Value, setting.Type)
		if err != nil {
			return err
		}
	}

	log.Printf("Seeded %d settings\n", len(settings))
	return nil
}

func (s *SettingSeeder) Truncate(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE TABLE settings CASCADE")
	return err
}
