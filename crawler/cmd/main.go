package main

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/2Geigh/Herb/crawler/internal/database"
	"github.com/2Geigh/Herb/crawler/internal/helper"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type (
	webpage struct {
		ResponseBody string `json:"response_body"`

		Domain            helper.SecondAndTopLevelDomain `json:"domain"`
		Url               helper.Url                     `json:"helper.Url"`
		Title             string                         `json:"title"`
		Description       string                         `json:"description"`
		Text              string                         `json:"text"`
		Outneighbours     []helper.Url                   `json:"outneighbours"`
		Date_discovered   time.Time                      `json:"date_discovered"`
		Date_last_crawled time.Time                      `json:"date_last_crawled"`
	}

	queueOfPages struct {
		mu    sync.Mutex
		links []helper.Url
	}
)

const (
	CRAWLER_POLITENESS_INTERVAL       time.Duration = 8 * time.Second
	CRAWLER_MINIMUM_OLDNESS_THRESHOLD time.Duration = 2592000 * time.Second // 30 days
)

var (
	seed_urls = []helper.Url{
		"https://nicholasgarcia.com",
		"https://angeldolly.com/",
		"https://nyscyra.net/",
		"https://0xffff.one",
		"https://v2ex.com",
	}
	pagesQueue = queueOfPages{links: []helper.Url{}, mu: sync.Mutex{}}
)

func (q *queueOfPages) dequeue() helper.Url {
	q.mu.Lock()
	defer q.mu.Unlock()

	dequeued := q.links[0]
	q.links = q.links[1:]
	return dequeued
}

func (q *queueOfPages) enqueue(urls []helper.Url) {
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

		time.Sleep(CRAWLER_POLITENESS_INTERVAL / 2)
	}

	wg.Wait()
}

func crawl(seed_url helper.Url, queue *queueOfPages, crawler_id uint, iterator *uint, wg *sync.WaitGroup) {
	defer wg.Done()

	queue.enqueue([]helper.Url{seed_url})

	for len(queue.links) > 0 {
		var (
			page webpage
		)

		currentUrl := queue.dequeue()

		// IF THIS URL'S DOMAIN HAS BEEN HIT
		// WITHIN THE LAST CRAWLER_POLITENESS_SLEEP_TIME (15 seconds):

		// IF THIS URL HAS BEEN HIT
		// WITHIN THE LAST CRAWLER_MINIMUM_OLDNESS_THRESHOLD (30 days):
		// continue
		if hasDomainBeenCrawledTooRecently(currentUrl, CRAWLER_POLITENESS_INTERVAL) {
			time.Sleep(CRAWLER_POLITENESS_INTERVAL)
		}

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

		body := struct {
			asReadCloser io.ReadCloser
			asBytes      []byte
		}{
			asReadCloser: response.Body,
		}

		body.asBytes, err = io.ReadAll(response.Body)
		if err != nil {
			log.Printf("[%s] read response body failed: %v", currentUrl, err)
			continue
		}
		page.ResponseBody = string(body.asBytes)

		if !utf8.Valid(body.asBytes) {
			log.Printf("[%s] invalid UTF-8", currentUrl)
			continue
		}

		doc, err := html.Parse(
			strings.NewReader(string(body.asBytes)),
		)
		if err != nil {
			log.Printf("[%s] parse HTML failed: %v", currentUrl, err)
			continue
		}

		page.Title = parsePageTitle(doc)
		page.Description = parsePageDescription(doc)

		page.Text, err = parsePageBody(doc)
		if err != nil {
			log.Printf("[%s] parse body content failed: %v", currentUrl, err)
			continue
		}

		hyperlinks := findHyperlinks(doc, currentUrl)
		page.Outneighbours = hyperlinks
		queue.enqueue(hyperlinks)

		log.Println()
		log.Println("crawler:       ", crawler_id)
		// log.Println("iter:          ", *iterator)
		log.Println("url:           ", currentUrl)
		log.Println("title:         ", page.Title)
		log.Println("desc:          ", page.Description)
		// log.Println("body:          ", len(page.Text), "bytes long")
		log.Println("outneighbours: ", len(page.Outneighbours))
		// log.Println("response_body: ", len(page.ResponseBody), "bytes long")
		log.Println("queue:         ", len(queue.links), "links long")

		*iterator += 1
	}

}

func hasDomainBeenCrawledTooRecently(link helper.Url, politeness_interval time.Duration) bool {
	// _, err := database.DB.Exec(
	// 	`SELECT last_crawled_date FROM pages WHERE `,
	// 	link.TrimTrailingSlash().GetSecondAndTopLevelDomain(),
	// )
	// if err == sql.ErrNoRows {

	// } else if err != nil {

	// }

	return true
}

func parsePageTitle(root_node *html.Node) string {
	var (
		pageTitle string
	)

	for node := range root_node.Descendants() {
		var (
			isTitle bool = node.DataAtom == atom.Title
			isH1    bool = node.DataAtom == atom.H1
			isH2    bool = node.DataAtom == atom.H2
			isH3    bool = node.DataAtom == atom.H3
		)

		if isTitle {
			if node.FirstChild != nil {
				pageTitle = strings.TrimSpace(node.FirstChild.Data)
				break
			}

			pageTitle = strings.TrimSpace(node.Data)
			break
		}

		if isH1 { // If no <title> found, use <h1> as fallback
			if node.FirstChild != nil {
				pageTitle = strings.TrimSpace(node.FirstChild.Data)
				break
			}

			pageTitle = strings.TrimSpace(node.Data)
			break
		}

		if isH2 { // If no <h1> found, use <h2> as fallback
			if node.FirstChild != nil {
				pageTitle = strings.TrimSpace(node.FirstChild.Data)
				break
			}

			pageTitle = strings.TrimSpace(node.Data)
			break
		}

		if isH3 { // If no <h2> found, use <h3> as fallback
			if node.FirstChild != nil {
				pageTitle = strings.TrimSpace(node.FirstChild.Data)
				break
			}

			pageTitle = strings.TrimSpace(node.Data)
			break
		}
	}

	return pageTitle
}

func parsePageDescription(root_node *html.Node) string {
	var (
		pageDescription string
	)

	for node := range root_node.Descendants() {
		var (
			isMeta bool = node.DataAtom == atom.Meta
		)
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

	return pageDescription
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

		if !isBodyText {
			continue
		}

		if node.FirstChild == nil {
			continue
		}

		if node.FirstChild.Type != html.TextNode {
			continue
		}

		// fmt.Println()
		// fmt.Println("               DATA", strings.TrimSpace(node.Data))
		// fmt.Println("           DATAATOM", node.DataAtom)
		// fmt.Println("           DATATYPE", node.Type)
		// fmt.Println("    FIRSTCHILD_DATA", strings.TrimSpace(node.FirstChild.Data))
		// fmt.Println("FIRSTCHILD_DATAATOM", node.FirstChild.DataAtom)
		// fmt.Println("FIRSTCHILD_DATATYPE", node.FirstChild.Type)

		_, err = sb.WriteString(fmt.Sprintf("%s ", strings.TrimSpace(node.FirstChild.Data)))
		if err != nil {
			err = fmt.Errorf("write to string builder failed: %w", err)
		}
	}

	return strings.TrimSpace(sb.String()), err
}

func findHyperlinks(root_node *html.Node, root_url helper.Url) []helper.Url {
	var (
		hyperlinks []helper.Url
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
				trimmedRootUrl helper.Url = root_url
				anchorHref     string     = attribute.Val
				newfoundLink   helper.Url
			)

			if len(anchorHref) < 2 {
				continue
			}

			if string(root_url[len(root_url)-1]) == "/" {
				trimmedRootUrl = root_url[0 : len(root_url)-1]
			}

			if string(anchorHref[0]) == "/" { // ex: <a href="/about">
				newfoundLink = helper.Url(string(trimmedRootUrl) + anchorHref)
			} else if len(anchorHref) >= len("http") &&
				string(anchorHref[0:len("http")]) != "http" { // ex: <a href="intro.html">
				newfoundLink = helper.Url(fmt.Sprintf("%s/%s", trimmedRootUrl, anchorHref))
			} else {
				newfoundLink = helper.Url(anchorHref)
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
			hyperlinks = append(hyperlinks, newfoundLink.TrimTrailingSlash())
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
