package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type (
	Webpage struct {
		ResponseBody string `json:"response_body"`

		FullDomain              Domain    `json:"full_domain"`
		TopAndSecondLevelDomain Domain    `json:"top_and_second_level_domain"`
		Url                     Url       `json:"Url"`
		Title                   string    `json:"title"`
		Description             string    `json:"description"`
		Text                    string    `json:"text"`
		Outneighbours           []Url     `json:"outneighbours"`
		Date_discovered         time.Time `json:"date_discovered"`
		Date_last_crawled       time.Time `json:"date_last_crawled"`
	}
)

func (page *Webpage) Save(db *sql.DB) error {

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("start tx failed: %w", err)
	}
	defer tx.Rollback()

	var (
		siteId           int64
		isSiteInDatabase bool = true
	)
	err = tx.QueryRow(
		`SELECT id 
		FROM sites
		WHERE full_domain = $1;`,
		page.FullDomain,
	).Scan(&siteId)
	if err == sql.ErrNoRows {
		isSiteInDatabase = false
	} else if err != nil {
		return fmt.Errorf("SELECT site id failed: %w", err)
	}

	var (
		pageId           int64
		isPageInDatabase bool = true
	)
	err = tx.QueryRow(
		`SELECT id 
		FROM pages
		WHERE link = $1;`,
		page.Url,
	).Scan(&pageId)
	if err == sql.ErrNoRows {
		isPageInDatabase = false
	} else if err != nil {
		return fmt.Errorf("SELECT page id failed: %w", err)
	}

	if !isSiteInDatabase {
		stmt, err := tx.Prepare(
			`INSERT INTO sites (
				second_and_top_level_domain,
				full_domain
			)
			VALUES ($1, $2)
			RETURNING id;`,
		)
		if err != nil {
			return fmt.Errorf("prepare INSERT site stmt failed: %w", err)
		}

		err = stmt.QueryRow(page.TopAndSecondLevelDomain, page.FullDomain).Scan(&siteId)
		if err != nil {
			return fmt.Errorf("execute INSERT site stmt failed: %w", err)
		}
	} else {
		_, err = tx.Exec(
			`UPDATE sites
			SET date_last_crawled = $1
			WHERE id = $2;`,
			time.Now(), siteId)
		if err != nil {
			return fmt.Errorf("update site date_last_crawled failed: %w", err)
		}
	}

	if !isPageInDatabase {
		stmt, err := tx.Prepare(
			`INSERT INTO pages (
				site_id,
				title,
				description,
				link,
				body_text,
				response_body
			)
			VALUES ($1, $2, $3, $4, $5, $6);`,
		)
		if err != nil {
			return fmt.Errorf("prepare INSERT page stmt failed: %w", err)
		}

		_, err = stmt.Exec(
			siteId,
			page.Title,
			page.Description,
			page.Url.TrimTrailingSlash(),
			page.Text,
			page.ResponseBody,
		)
		if err != nil {
			return fmt.Errorf("execute INSERT page stmt failed: %w", err)
		}
	} else {
		_, err = tx.Exec(
			`UPDATE pages
			SET date_last_crawled = $1
			WHERE id = $2;`,
			time.Now(), pageId)
		if err != nil {
			return fmt.Errorf("update page date_last_crawled failed: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx failed: %w", err)
	}

	return nil
}

func (p *Webpage) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &p)
}

func (p *Webpage) Value() (driver.Value, error) {
	return json.Marshal(p)
}
