package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type chunkedCSVReader struct {
	maxChunkSize     int
	currentChunkSize int

	// header of the CSV file, attached to every chunk.
	header      []string
	writeHeader bool

	r *csv.Reader
	// readErr contains the last error we got from a call to r.Read
	readErr error
	// lookahead, contains the next row from r.
	nextRow []string

	writeBuf *bytes.Buffer
}

// NewChunkedCSVReader creates a chunkedCSVReader based on the given input file and chunkSize. Note
// that chunkSize in not the maximum size that a chunk might grow to, but instead defines the
// "cutoff", which after reaching determines the end of a chunk. Effectively, this means that a
// chunk has a max size of chunkSize + size of 1 record.
func NewChunkedCSVReader(csvFile io.Reader, chunkSize int) (*chunkedCSVReader, error) {
	r := csv.NewReader(csvFile)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("could not read header: %w", err)
	}

	// now that we've read the header, reuse the slice for all future rows / reads.
	r.ReuseRecord = true

	// calculate the minimum chunkSize so that we can fit at least the header + one record.
	// TODO: do we need csv.NewWriter for correct size calculation? I'm assuming that at least for
	// the header, we usually won't have special chars and the likes, so maybe this is good enough?
	if l := len(strings.Join(header, ",")); l > chunkSize {
		// one for newline and one more so that we include at least one record.
		chunkSize = l + 2
	}

	return &chunkedCSVReader{
		header:   header,
		r:        r,
		writeBuf: new(bytes.Buffer),

		maxChunkSize: chunkSize,
	}, nil
}

// NextChunk returns true if there is another chunk to read, returning false if the underlying
// io.Reader has returned io.EOF.
func (c *chunkedCSVReader) NextChunk() bool {
	c.currentChunkSize = 0
	c.writeBuf.Reset()

	if c.readErr == io.EOF || (c.nextRow == nil && !c.nextLine()) {
		return false
	}

	c.writeHeader = true
	return true
}

// nextLine populates c.nextRow, returning true if there is a line to read.
func (c *chunkedCSVReader) nextLine() bool {
	var err error
	c.nextRow, err = c.r.Read()
	if err != nil {
		c.readErr = err
		return false
	}
	return true
}

func (c *chunkedCSVReader) writeNextRow(w *csv.Writer) error {
	if c.writeHeader {
		if err := w.Write(c.header); err != nil {
			return fmt.Errorf("could not write header to temporary CSV buffer: %w", err)
		}
		c.writeHeader = false
	}

	if err := w.Write(c.nextRow); err != nil {
		return fmt.Errorf("could not write row to temporary CSV buffer: %w", err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("could not flush temporary CSV buffer: %w", err)
	}
	return nil
}

// Read implements io.Reader, returning io.EOF once the chunkSize is reached. To continue reading
// the next chunk after receiving an io.EOF, call `NextChunk`.
func (c *chunkedCSVReader) Read(p []byte) (n int, err error) {
	if c.currentChunkSize >= c.maxChunkSize {
		return c.writeBuf.Read(p)
	}

	w := csv.NewWriter(c.writeBuf)
	// 1) the current writeBuf contents must be *less than* the size of p. This ensures we don't
	// grow the writeBuf too much over p to save memory (else we'd might populate it until
	// chunkSize before writing all contents to p).
	// 2) the current chunk must still have capacity for what's in the writeBuf. This doesn't stop
	// us from creating larger chunks than c.maxChunkSize, but guarantees that we write at max one
	// record more than chunkSize would allow us to.
	// 3) there needs to be no read error.
	for len(p) > c.writeBuf.Len() && (c.maxChunkSize-c.currentChunkSize) > c.writeBuf.Len() && c.readErr == nil {
		if err := c.writeNextRow(w); err != nil {
			return 0, err
		}
		// stop if there's no next line anymore.
		if !c.nextLine() {
			break
		}
	}
	if c.readErr != nil && c.readErr != io.EOF {
		return n, c.readErr
	}

	n, err = c.writeBuf.Read(p)
	if err != nil {
		return n, err
	}

	c.currentChunkSize += n

	return n, err
}
