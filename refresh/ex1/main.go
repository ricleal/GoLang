package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// The Basics (Types & Interfaces)

// Define a LogEntry struct: Include fields like Timestamp, LogLevel, Path, and ResponseTime.

// Create a Parser interface: Define a method Parse(line string) (*LogEntry, error).

// Implement a JSONParser and a RegexParser: This will refresh your memory on how Go's implicit interfaces work.

type JSONLogEntry struct {
	Time  time.Time `json:"time"`
	Level string    `json:"level"`
	Path  string    `json:"path"`
	Msg   string    `json:"msg"`
}

func (j *JSONLogEntry) String() string {
	b, err := json.Marshal(j)
	if err != nil {
		panic("Marshaling error")
	}
	s := string(b)
	return s
}

// Flat Log Entry Example
type FlatLogEntry struct {
	Time  time.Time
	Level string
	Path  string
	Msg   string
}

func (f *FlatLogEntry) String() string {
	s := fmt.Sprintf("%v [%s] %s path=%s", f.Time.Format("2006-01-02-15:04:05"), f.Level, f.Msg, f.Path)
	return s
}

// LogEntry interface
type LogEntry interface {
	String() string
}

// Parser interface
type Parser interface {
	Parse(line string) ([]LogEntry, error)
}

type JSONParser struct{}

func (j JSONParser) Parse(s string) ([]LogEntry, error) {
	var logEntry JSONLogEntry
	var logEntries []LogEntry
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		sReader := strings.NewReader(line)
		err := json.NewDecoder(sReader).Decode(&logEntry)
		if err != nil {
			return nil, err
		}
		logEntries = append(logEntries, &logEntry)
	}
	return logEntries, nil
}

func parse(s string, p Parser) {
	ll, err := p.Parse(s)
	if err != nil {
		panic("error in parser")
	}
	for _, l := range ll {
		fmt.Println(l)
	}
}

func main() {
	var lines string
	var i int64
	for i = 0; i < 10; i++ {
		j := JSONLogEntry{
			Time: time.Now().Add(time.Duration(i) * time.Hour), Level: "DEBUG",
			Path: "/tmp/ccc.ccc", Msg: "this is debug message",
		}
		lines += "\n" + j.String()
	}
	fmt.Println(lines)

	fmt.Println("-------------------------")

	var p JSONParser
	parse(lines, p)
}
