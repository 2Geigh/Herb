package main

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
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
		Body               string    `json:"body"`
		Title              string    `json:"title"`
		Description        string    `json:"description"`
		Outneighbours      []url     `json:"outneighbours"`
		Date_discovered    time.Time `json:"date_discovered"`
		Date_last_accessed time.Time `json:"date_last_accessed"`
	}

	queueOfPages struct {
		mu    sync.Mutex
		links []url
	}
)

const (
	CRAWLER_POLITENESS_SLEEP_TIME     time.Duration = 8 * time.Second
	CRAWLER_MINIMUM_OLDNESS_THRESHOLD time.Duration = 2592000 * time.Second // 30 days
)

var (
	seed_urls = []url{
		"https://nicholasgarcia.com",
		"https://angeldolly.com/",
		"https://nyscyra.net/",
		"https://0xffff.one",
		"https://v2ex.com",
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
		crwaler_id     uint = 0
		crawl_iterator uint = 0
		wg             sync.WaitGroup
	)

	err := database.InitializeDB()
	if err != nil {
		log.Fatalf("connect to database failed: %v", err)
	}
	defer database.DB.Close()

	for _, seed_url := range seed_urls {
		wg.Add(1)
		crwaler_id += 1

		go crawl(seed_url, &pagesQueue, crwaler_id, &crawl_iterator, &wg)

		time.Sleep(CRAWLER_POLITENESS_SLEEP_TIME / 2)
	}

	wg.Wait()
}

func crawl(seed_url url, queue *queueOfPages, crawler_id uint, iterator *uint, wg *sync.WaitGroup) {
	defer wg.Done()

	queue.enqueue([]url{seed_url})

	for len(queue.links) > 0 {
		var (
			page webpage
		)

		currentUrl := queue.dequeue()

		// IF THIS URL'S DOMAIN HAS BEEN HIT
		// WITHIN THE LAST CRAWLER_POLITENESS_SLEEP_TIME (15 seconds):
		time.Sleep(CRAWLER_POLITENESS_SLEEP_TIME)

		// IF THIS URL HAS BEEN HIT
		// WITHIN THE LAST CRAWLER_MINIMUM_OLDNESS_THRESHOLD (30 days):
		// continue

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

		page.Title, page.Description = parsePageMetadata(doc)

		page.Body, err = parsePageBody(doc)
		if err != nil {
			log.Printf("[%s] parse body content failed: %v", currentUrl, err)
			continue
		}

		hyperlinks := findHyperlinks(doc, currentUrl)
		queue.enqueue(hyperlinks)
		*iterator += 1

		log.Println()
		log.Println("id:    ", crawler_id)
		log.Println("iter:  ", *iterator)
		log.Println("url:   ", currentUrl)
		log.Println("title: ", page.Title)
		log.Println("desc:  ", page.Description)
		log.Println("body:  ", page.Body)
		log.Println("queue: ", len(queue.links), "links long")

		time.Sleep(CRAWLER_POLITENESS_SLEEP_TIME)
	}

}

func parsePageMetadata(root_node *html.Node) (string, string) {
	var (
		pageTitle       string
		pageDescription string
	)

	for node := range root_node.Descendants() {
		var (
			isTitle bool = node.DataAtom == atom.Title
			isH1    bool = node.DataAtom == atom.H1
			isH2    bool = node.DataAtom == atom.H2
			isH3    bool = node.DataAtom == atom.H3

			isMeta bool = node.DataAtom == atom.Meta
		)

		// No point in continuing to iterate over the loop
		// if everything has already been found
		if pageTitle != "" && pageDescription != "" {
			break
		}

		if isTitle && pageTitle == "" {
			pageTitle = strings.TrimSpace(node.FirstChild.Data)
		} else if isH1 && pageTitle == "" { // If no <title> found, use <h1> as fallback
			pageTitle = strings.TrimSpace(node.FirstChild.Data)
		} else if isH2 && pageTitle == "" { // If no <h1> found, use <h2> as fallback
			pageTitle = strings.TrimSpace(node.FirstChild.Data)
		} else if isH3 && pageTitle == "" { // If no <h2> found, use <h3> as fallback
			pageTitle = strings.TrimSpace(node.FirstChild.Data)
		}

		// Get page description
		if !isMeta {
			continue
		}
		var (
			isMetaDescription bool = slices.Contains(
				node.Attr,
				html.Attribute{
					Key: "name",
					Val: "description"},
			)
		)
		if !isMetaDescription {
			continue
		}
		var (
			contentIndex = slices.IndexFunc(
				node.Attr,
				func(attr html.Attribute) bool {
					return attr.Key == "content"
				},
			)
			hasContentKey bool = contentIndex != -1
		)
		if !hasContentKey {
			continue
		}
		pageDescription = strings.TrimSpace(node.Attr[contentIndex].Val)
		break
	}

	return pageTitle, pageDescription
}

func parsePageBody(root_node *html.Node) (string, error) {
	var (
		body_node *html.Node

		sb  strings.Builder
		err error
	)

	// Find <body> node
	for node := range root_node.Descendants() {
		if node.DataAtom != atom.Body {
			continue
		}

		body_node = node
		break
	}

	if body_node == nil {
		return sb.String(), fmt.Errorf("no <body> node found")
	}

	for node := range body_node.Descendants() {
		var (
			isBodyText = node.DataAtom != atom.Script &&
				node.DataAtom != atom.Style &&

				node.DataAtom != atom.Math &&
				node.DataAtom != atom.Embed &&
				node.DataAtom != atom.Iframe &&
				node.DataAtom != atom.Object &&
				node.DataAtom != atom.Picture &&
				node.DataAtom != atom.Source &&

				node.DataAtom != atom.Area &&
				node.DataAtom != atom.Audio &&
				node.DataAtom != atom.B &&
				node.DataAtom != atom.Canvas &&
				node.DataAtom != atom.Command &&
				node.DataAtom != atom.I &&
				node.DataAtom != atom.Img &&
				node.DataAtom != atom.Map &&
				node.DataAtom != atom.Svg &&
				node.DataAtom != atom.Track &&
				node.DataAtom != atom.Video &&

				node.Type != html.CommentNode
		)

		if node.FirstChild == nil {
			continue
		}

		if !isBodyText {
			continue
		}

		_, err = sb.WriteString(fmt.Sprintf("%s ", strings.TrimSpace(node.FirstChild.Data)))
		if err != nil {
			err = fmt.Errorf("write to string builder failed: %w", err)
		}
	}
	return strings.TrimSpace(sb.String()), err
}

// return strings.TrimSpace(sb.String()), err

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
