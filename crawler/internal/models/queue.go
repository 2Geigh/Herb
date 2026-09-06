package models

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
)

type (
	QueueOfPages struct {
		Mu    sync.Mutex
		Links []Url
	}
)

func (q *QueueOfPages) Cleanup(db *sql.DB) error {

	q.Mu.Lock()
	defer q.Mu.Unlock()

	result, err := db.Exec(
		`DELETE FROM link_queue a
		USING link_queue b
		WHERE a.hyperlink = b.hyperlink;`,
	)
	if err != nil {
		return fmt.Errorf("execute statement failed: %w", err)
	}

	duplicatesDeleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get result.RowsAffected() failed: %w", err)
	}

	log.Printf("removed %d duplicates from hyperlink queue", duplicatesDeleted)
	return err
}

func (q *QueueOfPages) Dequeue(db *sql.DB) (Url, error) {
	var (
		queueRow struct {
			url Url
			id  int64
		}
	)

	q.Mu.Lock()
	defer q.Mu.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return queueRow.url, fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	row := tx.QueryRow(
		`SELECT id, hyperlink 
		FROM link_queue 
		LIMIT 1;`,
	)
	err = row.Scan(&queueRow.id, &queueRow.url)
	if err == sql.ErrNoRows {
		return queueRow.url, fmt.Errorf("queue empty")
	} else if err != nil {
		return queueRow.url, fmt.Errorf("SELECT query failed: %w", err)
	}

	_, err = tx.Exec(`DELETE FROM link_queue WHERE id = $1;`, queueRow.id)
	if err != nil {
		return queueRow.url, fmt.Errorf("DELETE query failed: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return queueRow.url, fmt.Errorf("commit transaction failed: %w", err)
	}

	return queueRow.url, nil
}

func (q *QueueOfPages) Enqueue(urls []Url, db *sql.DB) error {
	q.Mu.Lock()
	defer q.Mu.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	for _, url := range urls {
		stmt, err := tx.Prepare(
			`INSERT INTO link_queue (hyperlink) VALUES ($1);`,
		)
		if err != nil {
			return fmt.Errorf("prepare statement failed: %w", err)
		}

		_, err = stmt.Exec(url)
		if err != nil {
			return fmt.Errorf("execute statement failed: %w", err)
		}

		err = stmt.Close()
		if err != nil {
			return fmt.Errorf("close statement failed: %w", err)
		}
	}

	// SAVE URL TO QUEUE TABLE IF NOT ALREADY PRESENT

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	return nil
}
