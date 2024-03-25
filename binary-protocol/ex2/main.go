package main

import (
	"encoding/binary"
	"fmt"
	"time"
)

// Cell represents a single cell in a binary packet (16 bytes)
// 0       1       2       3       4       5       6       7
// 0123456701234567012345670123456701234567012345670123456701234567
// +-------+-------+-------+-------+-------+-------+-------+------+
// |    SensorID   |   LocationID  |            Timestamp         |
// +-------+-------+-------+-------+-------+-------+-------+------+
// | Type1 |     Value1    | Type2 |     Value2    | Type3 |Value3|
// +-------+-------+-------+-------+-------+-------+-------+------+

type Cell struct {
	SensorID   uint16
	LocationID uint16
	Timestamp  uint32
	Type1      uint8
	Value1     uint16
	Type2      uint8
	Value2     uint16
	Type3      uint8
	Value3     uint8
}

// cellInput is a sample input for
var cellInput = [128]byte{
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
	0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
	0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
	0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
	0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f,
	0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47,
	0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f,
	0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57,
	0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
	0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67,
	0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f,
	0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
	0x78, 0x79, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f,
}

func decodeCell(data []byte) (Cell, error) {
	// decode a slice of packet
	var cell Cell
	cell.SensorID = binary.BigEndian.Uint16(data[0:2])
	cell.LocationID = binary.BigEndian.Uint16(data[2:4])
	cell.Timestamp = binary.BigEndian.Uint32(data[4:8])
	cell.Type1 = data[8]
	cell.Value1 = binary.BigEndian.Uint16(data[9:11])
	cell.Type2 = data[11]
	cell.Value2 = binary.BigEndian.Uint16(data[12:14])
	cell.Type3 = data[14]
	cell.Value3 = data[15]
	return cell, nil
}

func encodeCell(cell Cell) []byte {
	// encode a slice of packet
	buf := make([]byte, 16)
	binary.BigEndian.PutUint16(buf[0:], cell.SensorID)
	binary.BigEndian.PutUint16(buf[2:], cell.LocationID)
	binary.BigEndian.PutUint32(buf[4:], cell.Timestamp)
	buf[8] = cell.Type1
	binary.BigEndian.PutUint16(buf[9:], cell.Value1)
	buf[11] = cell.Type2
	binary.BigEndian.PutUint16(buf[12:], cell.Value2)
	buf[14] = cell.Type3
	buf[15] = cell.Value3
	return buf
}

// printCell prints the cell in a human-readable format
func printCell(cell Cell) {
	// print the cell
	fmt.Printf("SensorID:   %d\n", cell.SensorID)
	fmt.Printf("LocationID: %d\n", cell.LocationID)
	fmt.Printf("Time:       %s\n", time.Unix(int64(cell.Timestamp), 0))
	fmt.Printf("Type1: %2d, Value1: %d\n", cell.Type1, cell.Value1)
	fmt.Printf("Type2: %2d, Value2: %d\n", cell.Type2, cell.Value2)
	fmt.Printf("Type3: %2d, Value3: %d\n", cell.Type3, cell.Value3)
	fmt.Println("😻")
}

func main() {
	cellSize := 16 // 16 bytes
	cells := make([]Cell, len(cellInput)/cellSize)

	for i := 0; i < len(cellInput); i += cellSize {
		cell, err := decodeCell(cellInput[i : i+cellSize])
		printCell(cell)
		if err != nil {
			panic(err)
		}
		cells[i/cellSize] = cell
	}

	// let's encode the cells
	var encodedCells [128]byte
	for i, cell := range cells {
		copy(encodedCells[i*cellSize:], encodeCell(cell))
	}

	// Make sure the encoded data is the same as the original data
	for i := 0; i < len(cellInput); i++ {
		if cellInput[i] != encodedCells[i] {
			fmt.Printf("Mismatch at %d: %x != %x\n", i, cellInput[i], encodedCells[i])
		}
	}
}
