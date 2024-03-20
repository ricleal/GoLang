package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const URL = "https://txmchallenge-json.netlify.app/0.json"

// ["http://txmchallenge-json.netlify.app/0.json", "http://txmchallenge-json.netlify.app/1.json", "http://txmchallenge-json.netlify.app/2.json", "http://txmchallenge-json.netlify.app/3.json", "http://txmchallenge-json.netlify.app/4.json", "http://txmchallenge-json.netlify.app/5.json", "http://txmchallenge-json.netlify.app/6.json", "http://txmchallenge-json.netlify.app/7.json", "http://txmchallenge-json.netlify.app/8.json", "http://txmchallenge-json.netlify.app/9.json", "http://txmchallenge-json.netlify.app/10.json", "http://txmchallenge-json.netlify.app/11.json", "http://txmchallenge-json.netlify.app/12.json", "http://txmchallenge-json.netlify.app/13.json", "http://txmchallenge-json.netlify.app/14.json", "http://txmchallenge-json.netlify.app/15.json", "http://txmchallenge-json.netlify.app/16.json", "http://txmchallenge-json.netlify.app/17.json", "http://txmchallenge-json.netlify.app/18.json", "http://txmchallenge-json.netlify.app/19.json", "http://txmchallenge-json.netlify.app/20.json", "https://txmchallenge-json.netlify.app/21.json"]

type Crawler struct {
	visited map[string]struct{}
}

func NewCrawler() *Crawler {
	return &Crawler{
		visited: make(map[string]struct{}),
	}
}

func (c *Crawler) getURLs(url string) ([]string, error) {
	// Get the content of the URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get the content of the URL: %v", err)
	}
	defer resp.Body.Close()
	var data []string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode the content of the URL: %v", err)
	}
	return data, err
}

func (c *Crawler) Crawl(url string) {
	urlsCrawled := []string{url}
	var pos int
	for {
		if pos == len(urlsCrawled) {
			break
		}

		url = urlsCrawled[pos]
		if _, ok := c.visited[url]; ok {
			pos++
			continue
		}
		// check if the URL starts with https
		if url[:5] != "https" {
			pos++
			continue
		}
		c.visited[url] = struct{}{}
		// print the URL
		log.Println(url)
		urls, err := c.getURLs(url)
		if err != nil {
			log.Printf("FYI: failed to get the URLs: %v", err)
		}
		urlsCrawled = append(urlsCrawled, urls...)
		pos++
	}
}

func main() {
	c := NewCrawler()
	c.Crawl(URL)
}
