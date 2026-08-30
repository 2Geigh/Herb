package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/2Geigh/Herb/crawler/internal/database"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type (
	domain string
	url    string

	webpage struct {
		Domain             domain    `json:"domain"`
		Url                url       `json:"url"`
		Title              string    `json:"title"`
		Date_discovered    time.Time `json:"date_discovered"`
		Date_last_accessed time.Time `json:"date_last_accessed"`
	}

	queue struct {
		mu    sync.Mutex
		links []url
	}
)

func (q *queue) dequeue() url {
	q.mu.Lock()
	defer q.mu.Unlock()

	dequeued := q.links[0]
	q.links = q.links[1:]
	return dequeued
}

func (q *queue) enqueue(urls []url) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.links = append(q.links, urls...)
}

func main() {
	var (
		seed_urls = []url{"https://nicholasgarcia.com", "https://angeldolly.com/"}
		wg        sync.WaitGroup
	)

	err := database.InitializeDB()
	if err != nil {
		log.Fatalf("connect to database failed: %v", err)
	}
	defer database.DB.Close()

	for _, seed_url := range seed_urls {
		wg.Add(1)
		go crawl(seed_url)
	}

	wg.Wait()
}

func crawl(seed_url url) {
	log.Println("Crawling", seed_url)

	response, err := http.Get(string(seed_url))
	if err != nil {
		log.Printf("[%s] GET request failed: %v", seed_url, err)
	}
	defer response.Body.Close()

	doc, err := html.Parse(response.Body)
	if err != nil {
		log.Printf("[%s] parse HTML failed: %v", seed_url, err)
	}

	// save page title

	// save page body content

	hyperlinks := findHyperlinks(doc, seed_url)
	fmt.Println(hyperlinks)
}

func findHyperlinks(root_node *html.Node, root_url url) []url {
	var (
		hyperlinks []url
	)

	for n := range root_node.Descendants() {
		isAnchor :=
			n.Type == html.ElementNode &&
				n.DataAtom == atom.A

		if !isAnchor {
			continue
		}

		for _, a := range n.Attr {
			if a.Key != "href" {
				continue
			}

			var (
				trimmedRootUrl url    = root_url
				anchorHref     string = a.Val
				newfoundLink   url
			)

			if len(anchorHref) < 2 {
				break
			}

			if string(root_url[len(root_url)-1]) == "/" {
				trimmedRootUrl = root_url[0 : len(root_url)-2]
			}

			if string(anchorHref[0]) == "/" { // ex: <a href="/about">
				newfoundLink = url(string(trimmedRootUrl) + anchorHref)
			} else if string(anchorHref[0:4]) != "http" { // ex: <a href="intro.html">
				newfoundLink = url(fmt.Sprintf("%s/%s", trimmedRootUrl, anchorHref))
			} else {
				newfoundLink = url(anchorHref)
			}

			log.Printf("[%s] Found: %v", root_url, newfoundLink)
			hyperlinks = append(hyperlinks, newfoundLink)
			break
		}
	}

	return hyperlinks
}

func saveToDB(page webpage) error {
	_, err := database.DB.Exec(`INSERT INTO Pages $1`, page)
	if err != nil {
		return fmt.Errorf("insert page into database failed: %w", err)
	}

	return nil
}
