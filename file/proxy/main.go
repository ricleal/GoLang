package main

// To test:
// http --proxy="http:http://127.0.0.1:8080" "jsonplaceholder.typicode.com/posts/1"                                                                            ──(Thu,Mar21)─┘
// curl --proxy "http://127.0.0.1:8080" "jsonplaceholder.typicode.com/posts/1"
// 10 parallel requests:
// for i in $(seq 10); do http --proxy="http:http://127.0.0.1:8080" "jsonplaceholder.typicode.com/posts/1"&; done

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	PORT        = 8080
	logFilePath = "/tmp/proxy.log"
)

type LogStore struct {
	mu   sync.RWMutex
	file *os.File
}

func NewLogStore(filepath string) *LogStore {
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal("failed to open file:", err)
	}
	return &LogStore{
		file: file,
	}
}

func (l *LogStore) WriteLog(str string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.file.WriteString(str + "\n"); err != nil {
		log.Println("failed to write to file:", err)
	}
}

type Proxy struct {
	client   *http.Client
	logStore *LogStore
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleRequest")

	// forward request to the server
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// copy headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	// copy body
	io.Copy(w, resp.Body)

	// 27.218.230.166 - - [2024-03-06 16:53:13] "DELETE https://www.bolton-heath.org/homepage/ HTTP/1.1" 200 348

	str := logString(resp, r)
	fmt.Println(str)

	p.logStore.WriteLog(str)
}

func logString(resp *http.Response, r *http.Request) string {
	dateFormat := "2006-01-02 15:04:05"
	dateParsed, err := time.Parse(time.RFC1123, resp.Header.Get("Date"))
	if err != nil {
		log.Printf("failed to parse date, using now: %v", err)
		dateParsed = time.Now()
	}
	dateFormatted := dateParsed.Format(dateFormat)
	str := fmt.Sprintf("%s - - [%s] \"%s %s %s\" %d %d",
		r.RemoteAddr,
		dateFormatted,
		r.Method,
		r.URL.String(),
		r.Proto,
		resp.StatusCode,
		r.ContentLength,
	)
	return str
}

func NewProxy() *Proxy {
	return &Proxy{
		client:   &http.Client{},
		logStore: NewLogStore(logFilePath),
	}
}

func main() {
	fmt.Printf("Listening on :%d...\n", PORT)
	proxy := NewProxy()

	if err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), proxy); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
