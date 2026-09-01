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

	queueOfPages struct {
		mu    sync.Mutex
		links []url
	}
)

const (
	CRAWLER_POLITENESS_SLEEP_TIME = 15 * time.Second
)

var (
	seed_urls = []url{
		// "https://nicholasgarcia.com",
		"https://angeldolly.com/",
		"https://nyscyra.net/",
		"https://0xffff.one",
		// "https://v2ex.com",
	}
	pagesQueue = queueOfPages{links: []url{}, mu: sync.Mutex{}}
)

func (q *queueOfPages) dequeue() url {
	q.mu.Lock()
	defer q.mu.Unlock()

	dequeued := q.links[0]
	q.links = q.links[1:]
	return dequeued
}

func (q *queueOfPages) enqueue(urls []url) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.links = append(q.links, urls...)
}

func main() {
	var (
		wg sync.WaitGroup
	)

	err := database.InitializeDB()
	if err != nil {
		log.Fatalf("connect to database failed: %v", err)
	}
	defer database.DB.Close()

	for _, seed_url := range seed_urls {
		wg.Add(1)
		go crawl(seed_url, &pagesQueue, &wg)
		time.Sleep(CRAWLER_POLITENESS_SLEEP_TIME / 2)
	}

	wg.Wait()
}

func crawl(seed_url url, queue *queueOfPages, wg *sync.WaitGroup) {
	defer wg.Done()

	queue.enqueue([]url{seed_url})

	for len(queue.links) > 0 {
		var (
			page webpage
		)

		currentUrl := queue.dequeue()
		log.Println("url:   ", currentUrl)

		response, err := http.Get(string(currentUrl))
		if err != nil {
			log.Printf("[%s] GET request unfulfilled: %v", currentUrl, err)
			continue
		}
		defer response.Body.Close()

		var (
			isRequestSuccessful bool = response.StatusCode >= 200 && response.StatusCode < 300
		)
		if !isRequestSuccessful {
			log.Printf("[%s] request unsuccessful: %s", currentUrl, response.Status)
			continue
		}

		doc, err := html.Parse(response.Body)
		if err != nil {
			log.Printf("[%s] parse HTML failed: %v", currentUrl, err)
			continue
		}

		// save page title
		page.Title = getPageTitle(doc)

		// save page body content

		hyperlinks := findHyperlinks(doc, currentUrl)
		queue.enqueue(hyperlinks)

		log.Println("title: ", page.Title)
		log.Println("queue: ", len(queue.links), "links long")
		log.Println()

		time.Sleep(CRAWLER_POLITENESS_SLEEP_TIME)
	}

}

func getPageTitle(root_node *html.Node) string {
	for node := range root_node.Descendants() {
		var (
			isTitle bool = node.DataAtom == atom.Title
			isH1    bool = node.DataAtom == atom.H1
			isH2    bool = node.DataAtom == atom.H2
			isH3    bool = node.DataAtom == atom.H3
		)

		if isTitle {
			return node.FirstChild.Data
		}

		// If no <title> found, use <h1> as fallback
		if isH1 {
			return node.FirstChild.Data
		}

		// If no <h1> found, use <h2> as fallback
		if isH2 {
			return node.FirstChild.Data
		}

		// If no <h2> found, use <h3> as fallback
		if isH3 {
			return node.FirstChild.Data
		}
	}

	return ""
}

func findHyperlinks(root_node *html.Node, root_url url) []url {
	var (
		hyperlinks []url
	)

	for node := range root_node.Descendants() {
		isAnchor :=
			node.Type == html.ElementNode &&
				node.DataAtom == atom.A

		if !isAnchor {
			continue
		}

		for _, attribute := range node.Attr {
			if attribute.Key != "href" {
				continue
			}

			var (
				trimmedRootUrl url    = root_url
				anchorHref     string = attribute.Val
				newfoundLink   url
			)

			if len(anchorHref) < 2 {
				continue
			}

			if string(root_url[len(root_url)-1]) == "/" {
				trimmedRootUrl = root_url[0 : len(root_url)-1]
			}

			if string(anchorHref[0]) == "/" { // ex: <a href="/about">
				newfoundLink = url(string(trimmedRootUrl) + anchorHref)
			} else if len(anchorHref) >= len("http") &&
				string(anchorHref[0:len("http")]) != "http" { // ex: <a href="intro.html">
				newfoundLink = url(fmt.Sprintf("%s/%s", trimmedRootUrl, anchorHref))
			} else {
				newfoundLink = url(anchorHref)
			}

			if len(newfoundLink) < len("http://") {
				continue
			}

			var (
				isHttpUriScheme        bool = len(anchorHref) >= len("http") && (anchorHref[0:len("http")] == "http")
				isHttpsUriScheme       bool = len(anchorHref) >= len("https") && (anchorHref[0:len("https")] == "https")
				isAlternativeUriScheme bool = !isHttpUriScheme && !isHttpsUriScheme
			)
			if isAlternativeUriScheme {
				continue
			}

			// REMOVE ? QUERIES FROM URLs

			// REMOVE mailto: AND ANY OTHER SUCH TYPES OF URLS

			// log.Println()
			// log.Println("root_url", root_url)
			// log.Println("trimmedRootUrl", trimmedRootUrl)
			// log.Println("anchorHref", anchorHref)
			// log.Println("newfoundLink", newfoundLink)
			// log.Println("isAlternativeUriScheme", isAlternativeUriScheme)
			// log.Printf("[%s] Found: %v", root_url, newfoundLink)
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
