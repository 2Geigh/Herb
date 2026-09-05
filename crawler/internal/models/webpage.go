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

		Domain            SecondAndTopLevelDomain `json:"domain"`
		Url               Url                     `json:"Url"`
		Title             string                  `json:"title"`
		Description       string                  `json:"description"`
		Text              string                  `json:"text"`
		Outneighbours     []Url                   `json:"outneighbours"`
		Date_discovered   time.Time               `json:"date_discovered"`
		Date_last_crawled time.Time               `json:"date_last_crawled"`
	}
)

func (page *Webpage) Save(db *sql.DB) error {
	var (
		isNewlyDiscoveredSite bool
		isNewlyDiscoveredPage bool

		siteId int
		pageId int
	)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("start transaction failed: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
	}()

	// Verify if site is already recorded in database
	stmt, err := tx.Prepare(`SELECT id FROM sites WHERE second_and_top_level_domain = $1;`)
	if err != nil {
		return fmt.Errorf("prepare 'site' SELECT statement failed: %w", err)
	}
	defer stmt.Close()
	rows, err := stmt.Query(page.Domain)
	if err != nil {
		return fmt.Errorf("execute 'site' SELECT statement failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(siteId)
		if err != nil {
			return fmt.Errorf("scan returned SELECT site id failed: %w", err)
		}
		isNewlyDiscoveredSite = true
	}

	if isNewlyDiscoveredSite {
		stmt, err = tx.Prepare(`INSERT INTO Sites (second_and_top_level_domain) VALUES ($1);`)
		if err != nil {
			return fmt.Errorf("prepare 'site' INSERT statement failed: %w", err)
		}
		defer stmt.Close()

		result, err := stmt.Exec(page.Domain)
		if err != nil {
			return fmt.Errorf("execute 'site' INSERT transaction failed: %w", err)
		}

		var lastInsertId int64
		lastInsertId, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("parse returned INSERT site id failed: %w", err)
		}
		siteId = int(lastInsertId)
	} else {
		stmt, err = tx.Prepare(`UPDATE sites
			SET
				date_last_crawled = $1
			
			WHERE id = $2;`)
		if err != nil {
			return fmt.Errorf("prepare 'sites' UDPATE statement failed: %w", err)
		}
		defer stmt.Close()

		_, err := stmt.Exec(
			time.Now(),
			siteId,
		)
		if err != nil {
			return fmt.Errorf("execute 'sites' UDPATE transaction failed: %w", err)
		}
	}

	// Verify if page is already recorded in database
	stmt, err = tx.Prepare(`SELECT id FROM pages WHERE link = $1;`)
	if err != nil {
		return fmt.Errorf("prepare 'page' SELECT statement failed: %w", err)
	}
	defer stmt.Close()
	rows, err = stmt.Query(page.Url)
	if err != nil {
		return fmt.Errorf("execute 'page' SELECT statement failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(siteId)
		if err != nil {
			return fmt.Errorf("scan returned SELECT page id failed: %w", err)
		}
		isNewlyDiscoveredPage = true
	}

	if isNewlyDiscoveredPage {
		stmt, err = tx.Prepare(`INSERT INTO pages (
				title, 
				description,
				body_text,
				response_body,
			) 
			VALUES ($1, $2, $3, $4);`)
		if err != nil {
			return fmt.Errorf("prepare 'pages' INSERT statement failed: %w", err)
		}
		defer stmt.Close()

		result, err := stmt.Exec(
			page.Title,
			page.Description,
			page.Text,
			page.ResponseBody,
		)
		if err != nil {
			return fmt.Errorf("execute 'pages' INSERT transaction failed: %w", err)
		}

		var lastInsertId int64
		lastInsertId, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("parse returned INSERT page id failed: %w", err)
		}
		pageId = int(lastInsertId)
	} else {
		stmt, err = tx.Prepare(`UPDATE pages
			SET
				title = $1,
				description = $2,
				body_text = $3,
				response_body = $4,
				date_last_crawled = $5
			
			WHERE id = $6;`)
		if err != nil {
			return fmt.Errorf("prepare 'pages' UDPATE statement failed: %w", err)
		}
		defer stmt.Close()

		_, err := stmt.Exec(
			page.Title,
			page.Description,
			page.Text,
			page.ResponseBody,
			time.Now(),
			pageId,
		)
		if err != nil {
			return fmt.Errorf("execute 'pages' UDPATE transaction failed: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	return err
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
