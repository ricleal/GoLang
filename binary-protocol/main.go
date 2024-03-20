package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// 0       1       2       3       4       5       6       7
// 0123456701234567012345670123456701234567012345670123456701234567
// +-------+-------+-------+-------+-------+-------+-------+------+
// |    SensorID   |   LocationID  |            Timestamp         |
// +-------+-------+-------+-------+-------+-------+-------+------+
// |      Temp     |
// +---------------+
// The data is arranged in a fixed-size binary packet of 80 bits that divided into four fields
// including SensorID (16 bit), LocationID (16 bit), Timestamp (32 bit), and Temperature (16 bit).

type packet struct {
	SensorID    uint16
	LocationID  uint16
	Timestamp   uint32
	Temperature int16
}

func encodePackets(packets []packet) (io.Reader, error) {
	// encode a slice of packet
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, packets)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func decodePackets(data io.Reader, n int) ([]packet, error) {
	// decode a slice of packet
	packets := make([]packet, n)
	err := binary.Read(data, binary.LittleEndian, &packets)
	if err != nil {
		return nil, err
	}
	return packets, nil
}

func encodeSample() {
	buf := make([]byte, 10)
	now := time.Now()
	binary.BigEndian.PutUint16(buf[0:], 0xa20c)             // sensorID
	binary.BigEndian.PutUint16(buf[2:], 0x04af)             // locationID
	binary.BigEndian.PutUint32(buf[4:], uint32(now.Unix())) // timestamp
	binary.BigEndian.PutUint16(buf[8:], 479)                // temp

	fmt.Printf("%x\n", buf)
	fmt.Printf("%v\n", buf)
}

func main() {
	encodeSample()

	packets := []packet{
		{65535, 65534, 4294967295, 32767},
		{65533, 65532, 4294967294, -32768},
		{0x0001, 0x0002, 0x0003, 0x0004},
		{0x0005, 0x0006, 0x0007, 0x0008},
	}
	fmt.Println("Original packets")
	for _, p := range packets {
		fmt.Println(p.SensorID, p.LocationID, p.Timestamp, p.Temperature)
	}
	data, err := encodePackets(packets)
	if err != nil {
		panic(err)
	}
	packets, err = decodePackets(data, len(packets))
	if err != nil {
		panic(err)
	}
	fmt.Println("Decoded packets")
	for _, p := range packets {
		fmt.Println(p.SensorID, p.LocationID, p.Timestamp, p.Temperature)
	}
}
