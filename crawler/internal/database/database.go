package database

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

type (
	RankedDomain struct {
		Position int     `json:"position"`
		Domain   string  `json:"domain"`
		Count    int     `json:"count"`
		Etv      float64 `json:"etv"`
	}
)

var (
	DB *sql.DB = nil

	//go:embed migrations/*.sql
	embedMigrations embed.FS

	//go:embed data/*.json
	embedData embed.FS
)

func InitializeDomainBlacklist(db *sql.DB) error {
	var (
		topThousandDomains = struct {
			asBytes  []byte
			asString string
			asJson   []RankedDomain
		}{}

		err error
	)

	topThousandDomains.asBytes, err = embedData.ReadFile("data/ranked_domains.json")
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	err = json.Unmarshal(topThousandDomains.asBytes, &topThousandDomains.asJson)
	if err != nil {
		return fmt.Errorf("unmarshal json failed: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	defer tx.Rollback()

	tx.Exec(`DELETE FROM domain_blacklist *;`)

	for _, entry := range topThousandDomains.asJson {
		stmt, err := tx.Prepare(
			`INSERT INTO domain_blacklist (domain) values ($1);`,
		)
		if err != nil {
			return fmt.Errorf("prepare stmt failed: %w", err)
		}

		_, err = stmt.Exec(entry.Domain)
		if err != nil {
			return fmt.Errorf("execute stmt failed: %w", err)
		}
		stmt.Close()
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx failed: %w", err)
	}

	fmt.Println("Domain blacklist initialized successfully")

	return nil
}

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

func ReportDatabaseHealth() {
	// for {
	stats := DB.Stats()
	log.Printf(`[DB STATS] InUse: %d | Idle: %d | Open: %d | WaitCount: %d`,
		stats.InUse, stats.Idle, stats.OpenConnections, stats.WaitCount)

	// time.Sleep(5 * time.Second)
	// }
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
