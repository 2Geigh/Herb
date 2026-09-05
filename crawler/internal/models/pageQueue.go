package models

import (
	"sync"
)

type (
	QueueOfPages struct {
		Mu    sync.Mutex
		Links []Url
	}
)

func (q *QueueOfPages) Dequeue() Url {
	q.Mu.Lock()
	defer q.Mu.Unlock()

	dequeued := q.Links[0]
	q.Links = q.Links[1:]
	return dequeued
}

func (q *QueueOfPages) Enqueue(urls []Url) {
	q.Mu.Lock()
	defer q.Mu.Unlock()

	q.Links = append(q.Links, urls...)
}
