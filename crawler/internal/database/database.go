package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

var (
	DB *sql.DB = nil

	//go:embed migrations/*.sql
	embedMigrations embed.FS
)

func InitializeDB() error {
	log.Println("Connecting to Postgresql...")

	var (
		username string = os.Getenv("DB_USERNAME")
		password string = os.Getenv("DB_PASSWORD")
		dbHost   string = os.Getenv("DB_HOST")
		dbPort   string = os.Getenv("DB_CONTAINER_PORT")
		dbName   string = os.Getenv("DB_NAME")

		dsn string = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable", username, password, dbHost, dbPort, dbName)

		err error
	)

	fmt.Println(dsn)

	if username == "" {
		log.Println("Warning: DB_USERNAME is empty. Connection might fail.")
	}

	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database connection failed: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("verify database connection failed: %w", err)
	}
	log.Println("Database connection successful.")

	log.Println("Executing migrations...")
	err = migrate(DB)
	if err != nil {
		return fmt.Errorf("database migration(s) failed: %w", err)
	}

	return nil
}

func migrate(db *sql.DB) error {

	goose.SetBaseFS(embedMigrations)

	err := goose.SetDialect("postgres")
	if err != nil {
		return fmt.Errorf("Goose: set database dialect failed: %w", err)
	}

	err = goose.Up(db, "migrations")
	if err != nil {
		return fmt.Errorf("Goose: apply migrations failed: %w", err)
	}

	return nil
}

func ReportDatabaseHealth() {
	// for {
	stats := DB.Stats()
	log.Printf(`[DB STATS] InUse: %d | Idle: %d | Open: %d | WaitCount: %d`,
		stats.InUse, stats.Idle, stats.OpenConnections, stats.WaitCount)

	// time.Sleep(5 * time.Second)
	// }
}
