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

type webpage struct {
	Url                string    `json:"url"`
	Title              string    `json:"title"`
	Date_discovered    time.Time `json:"date_discovered"`
	Date_last_accessed time.Time `json:"date_last_accessed"`
}

func main() {
	var (
		seed_urls = []string{"https://nicholasgarcia.com", "https://angeldolly.com/"}
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

func crawl(seed_url string) {
	log.Println("Crawling", seed_url)

	response, err := http.Get(seed_url)
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

	for n := range doc.Descendants() {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			for _, a := range n.Attr {
				if a.Key == "href" {

					var (
						trimmedSeedUrl string = seed_url
						anchorHref     string = a.Val
						newfoundLink   string
					)

					if len(anchorHref) < 2 {
						break
					}

					if string(seed_url[len(seed_url)-1]) == "/" {
						trimmedSeedUrl = seed_url[0 : len(seed_url)-2]
					}

					if string(anchorHref[0]) == "/" { // ex: <a href="/about">
						newfoundLink = trimmedSeedUrl + anchorHref
					} else if string(anchorHref[0:4]) != "http" { // ex: <a href="intro.html">
						newfoundLink = fmt.Sprintf("%s/%s", trimmedSeedUrl, anchorHref)
					} else {
						newfoundLink = anchorHref
					}

					log.Printf("[%s] Found: %v", seed_url, newfoundLink)
					break
				}
			}
		}
	}
}

func saveToDB(page webpage) {
	database.DB.Exec("insert into")
}
