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
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type CacheEntry struct {
	Body       []byte
	Expiration time.Time
}

type Cache struct {
	mc *memcache.Client
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	mc := memcache.New("localhost:11211")
	return &Cache{
		mc: mc,
	}
}

// Get returns the value for the given key
func (c *Cache) Get(key string) ([]byte, error) {
	item, err := c.mc.Get(key)
	if err != nil {
		return nil, err
	}
	return item.Value, nil
}

// Set sets the value for the given key
func (c *Cache) Set(key string, value []byte) error {
	item := &memcache.Item{
		Key:        key,
		Value:      value,
		Expiration: 0,
	}
	return c.mc.Set(item)
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

// NewCachingReverseProxy creates a new reverse proxy instance
// that caches the responses
// It uses the given cache instance to store the responses
// It uses the given targetURL to perform the actual requests
// It uses the default http.Transport to perform the requests
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
			cacheKey := makeCacheKey(response.Request)
			// Read the response body
			body, err := httputil.DumpResponse(response, true)
			if err != nil {
				return err
			}
			return cache.Set(cacheKey, body)
		},
	}
}

type cachingTransport struct {
	innerTransport http.RoundTripper
	cache          *Cache
}

// RoundTrip performs the actual request and caches the response
// if it is not already in the cache
// It also adds the X-Forwarded-For header to the request
// to indicate the original client IP address
func (c *cachingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if the response is in the cache
	cacheKey := makeCacheKey(req)
	if cachedResponse, err := c.cache.Get(cacheKey); err == nil {
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
	if err := http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil); err != nil {
		log.Fatal(err)
	}
}
