package models

import (
	"database/sql"
	"fmt"
	"time"
)

type (
	SecondAndTopLevelDomain string
)

func (d SecondAndTopLevelDomain) HasBeenRequestedTooRecently(politeness_interval time.Duration, queue *QueueOfPages, db *sql.DB) (bool, error) {
	var (
		lastCrawled time.Time

		// better to be too
		// polite than not nice enough
		// to an API
		hasBeenCrawledTooRecently bool = true
	)

	queue.Mu.Lock()
	defer queue.Mu.Unlock()

	stmt, err := db.Prepare(
		`SELECT date_last_crawled
		FROM sites
		WHERE second_and_top_level_domain = $1;`,
	)
	if err != nil {
		return hasBeenCrawledTooRecently, fmt.Errorf("prepare stmt failed: %w", err)
	}
	defer stmt.Close()

	err = stmt.QueryRow(d).Scan(&lastCrawled)
	if err == sql.ErrNoRows {
		hasBeenCrawledTooRecently = false
	} else if err != nil {
		return hasBeenCrawledTooRecently, fmt.Errorf("execute stmt failed: %w", err)
	}

	if time.Since(lastCrawled) > politeness_interval {
		hasBeenCrawledTooRecently = false
	}

	return hasBeenCrawledTooRecently, nil
}

func (d SecondAndTopLevelDomain) IsBlacklisted(db *sql.DB) (bool, error) {
	var (
		exists bool
	)

	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM domain_blacklist WHERE domain = $1);`,
		d,
	).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}

	return exists, nil
}
