package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"os"
)

const outFileName = "/tmp/reconstructed_file.csv"
const inFileName = "assemble-file/data.csv.gz"
const seed = 1234

type Chunk struct {
	Offset  int64
	Payload []byte
}

func gunzip(filename string) ([]byte, error) {
	// Open the input file
	inputFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(inputFile)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	// Read the gunzipped data
	data, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func main() {

	var chunks []*Chunk

	// Read the gunzipped data into memory just for the sake of this example
	// In a real-world scenario, you would never read the entire file into memory
	data, err := gunzip(inFileName)
	if err != nil {
		fmt.Printf("Failed to gunzip file: %s\n", err)
		return
	}

	// Create a bytes.Reader
	reader := bytes.NewReader(data)

	// Create a byte slice to store the read data
	buffer := make([]byte, 1024)

	// Read bytes until the end of the slice
	start := 0
	for {
		reader.Seek(int64(start), 0)
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// Reached the end of the slice
				break
			}
			fmt.Println("Failed to read slice:", err)
			return
		}

		// Process the read bytes (buffer[:n])
		c := &Chunk{
			Offset: int64(start),
		}
		c.Payload = make([]byte, n)
		copy(c.Payload, buffer[:n])
		chunks = append(chunks, c)
		start += n
	}

	// Shuffle the chunks
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(chunks), func(i, j int) { chunks[i], chunks[j] = chunks[j], chunks[i] })

	// Create the output file
	outputFile, err := os.Create(outFileName)
	if err != nil {
		fmt.Printf("Failed to create output file: %s\n", err)
		return
	}
	defer outputFile.Close()

	// Assemble the file from chunks
	for i := 0; i < len(chunks); i++ {
		chunk := chunks[i]
		err = writeChunk(outputFile, chunk)
		if err != nil {
			fmt.Printf("Failed to write chunk with offset %d: %s\n", chunk.Offset, err)
			return
		}
	}

	fmt.Println("File successfully reconstructed.")
}

// Function to write a chunk to the output file at the specified offset
func writeChunk(outputFile *os.File, chunk *Chunk) error {
	outputFile.Seek(chunk.Offset, 0)
	_, err := outputFile.WriteAt(chunk.Payload, chunk.Offset)
	if err != nil {
		return err
	}
	return nil
}
