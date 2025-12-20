package refresh

import (
	"fmt"
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

func (j *JSONLogEntry) String() *string {
	return j.String()
}

type FlatLogEntry struct {
	Time  time.Time
	Level string
	Path  string
	Msg   string
}

func (f *FlatLogEntry) String() *string {
	s := fmt.Sprintf("%v [%s] %s path=%s", f.Time, f.Level, f.Msg, f.Path)
	return &s
}

type LogEntry interface {
	String() *string
}

type Parser interface {
	Parse(line string) (*LogEntry, error)
}

type JSONParser struct{}
