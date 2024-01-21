package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type CacheEntry struct {
	Body       []byte
	Expiration time.Time
}

type Cache struct {
	mutex sync.Mutex
	cache map[string]CacheEntry
}

func NewCache() *Cache {
	return &Cache{
		cache: make(map[string]CacheEntry),
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.cache[key]
	if !exists || time.Now().After(entry.Expiration) {
		delete(c.cache, key)
		return nil, false
	}

	return entry.Body, true
}

func (c *Cache) Set(key string, body []byte, expiration time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[key] = CacheEntry{
		Body:       body,
		Expiration: expiration,
	}
}

func buildHash(body []byte) string {
	hasher := sha1.New()
	hasher.Write(body)
	return hex.EncodeToString(hasher.Sum(nil))
}

func makeCacheKey(req *http.Request) string {
	keyPrefix := req.Method + ":" + req.URL.String()
	return buildHash([]byte(keyPrefix))
}

func NewCachingReverseProxy(targetURL *url.URL, cache *Cache) *httputil.ReverseProxy {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
		},
		Transport: &cachingTransport{
			innerTransport: transport,
			cache:          cache,
		},
		ModifyResponse: func(response *http.Response) error {
			// Cache the response for a certain duration
			expiration := time.Now().Add(5 * time.Minute)
			cacheKey := makeCacheKey(response.Request)
			// Read the response body
			body, err := httputil.DumpResponse(response, true)
			if err != nil {
				return err
			}
			cache.Set(cacheKey, body, expiration)
			return nil
		},
	}
}

type cachingTransport struct {
	innerTransport http.RoundTripper
	cache          *Cache
}

func (c *cachingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if the response is in the cache
	cacheKey := makeCacheKey(req)
	if cachedResponse, ok := c.cache.Get(cacheKey); ok {
		log.Println("Serving from cache:", cacheKey)
		// Build the response from the cached response
		resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(cachedResponse)), req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	// Add X-Forwarded-For header
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	// If not in the cache, perform the actual request using the inner transport
	response, err := c.innerTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	log.Println("Adding to cache:", cacheKey)
	return response, nil
}

func main() {
	targetURL, _ := url.Parse("http://localhost:8081")
	cache := NewCache()

	proxy := NewCachingReverseProxy(targetURL, cache)

	http.Handle("/", proxy)

	port := 8080
	log.Printf("Listening on :%d...\n", port)
	http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil)
}
