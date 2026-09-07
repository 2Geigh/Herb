package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type (
	Domain string
)

func (d Domain) GetSecondAndTopLevelDomain() Domain {
	/*

		>>> getDomain(https://www.youtube.com/watch?v=dQw4w9WgXcQ)
		youtube.com

	*/

	domainWithoutProtocol := d.StripProtocol()

	domainWithoutRoutes, _, _ := strings.Cut(string(domainWithoutProtocol), "/")

	domainLevels := strings.Split(domainWithoutRoutes, ".")

	topLevel := domainLevels[len(domainLevels)-1]

	if len(topLevel) < 2 {
		return Domain(topLevel)
	}

	secondLevel := domainLevels[len(domainLevels)-2]

	return Domain(secondLevel + "." + topLevel)
}

func (d Domain) HasBeenRequestedTooRecently(politeness_interval time.Duration, queue *QueueOfPages, db *sql.DB) (bool, error) {
	var (
		secondAndTopLevelDomain = d.GetSecondAndTopLevelDomain()
		lastCrawled             time.Time

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

	err = stmt.QueryRow(secondAndTopLevelDomain).Scan(&lastCrawled)
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

func (d Domain) IsBlacklisted(db *sql.DB) (bool, error) {
	var (
		secondAndTopLevelDomain = d.GetSecondAndTopLevelDomain()

		exists bool
	)

	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM domain_blacklist WHERE domain = $1);`,
		secondAndTopLevelDomain,
	).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}

	return exists, nil
}

func (d Domain) StripProtocol() Domain {

	_, domainWithoutProtocol, includesProtocol := strings.Cut(string(d), "://")

	if !includesProtocol {
		return d
	}

	return Domain(domainWithoutProtocol)
}
