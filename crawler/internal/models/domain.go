package models

import (
	"database/sql"
	"fmt"
	"time"
)

type (
	SecondAndTopLevelDomain string
)

func (d SecondAndTopLevelDomain) HasBeenRequestedTooRecently(politeness_interval time.Duration, queue QueueOfPages, db *sql.DB) (bool, error) {
	var (
		lastCrawled time.Time

		// better to be too
		// polite than not polite
		// polite to websites
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
