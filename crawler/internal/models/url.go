package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type (
	Url string
)

func (url Url) GetDomain() Domain {
	/*

		>>> getDomain(https://www.youtube.com/watch?v=dQw4w9WgXcQ)
		youtube.com

	*/

	linkComponents := strings.Split(string(url), "://")
	// [0] == http:// || https://
	// [1] == www.example.com/thingy

	// protocol := linkComponents[0] + "://"
	domainWithoutRoutes, _, _ := strings.Cut(linkComponents[1], "/")

	_, domainWithoutWWW, containsWWW := strings.Cut(domainWithoutRoutes, "www.")
	if containsWWW {
		return Domain(domainWithoutWWW)
	}

	return Domain(domainWithoutRoutes)
}

func (url Url) IsTooRecentlyCrawled(db *sql.DB, oldness_threshold time.Duration) (bool, error) {
	var (
		lastCrawled          time.Time
		isTooRecentlyCrawled bool = false
	)

	stmt, err := db.Prepare(
		`SELECT date_last_crawled
		FROM pages
		WHERE link = $1;`,
	)
	if err != nil {
		return isTooRecentlyCrawled, fmt.Errorf("prepare stmt failed: %w", err)
	}
	defer stmt.Close()

	err = stmt.QueryRow(url.TrimTrailingSlash()).Scan(&lastCrawled)
	if err != nil && err != sql.ErrNoRows {
		return isTooRecentlyCrawled, fmt.Errorf("execute stmt failed: %w", err)
	}

	if time.Since(lastCrawled) < oldness_threshold {
		isTooRecentlyCrawled = true
	}

	return isTooRecentlyCrawled, nil
}

func (url Url) TrimTrailingSlash() Url {
	if string(url[len(url)-1]) != "/" {
		return url
	}

	return url[0 : len(url)-1]
}
