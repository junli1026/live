package rtmp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
)

const CHUNK_HEADER_MAX_SIZE = 18

type Chunk struct {
	Type     byte
	StreamId uint32
}

func ReadChunk(conn net.Conn) error {
	fmt.Println("here")
	reader := bufio.NewReader(conn)

	buf := make([]byte, CHUNK_HEADER_MAX_SIZE, CHUNK_HEADER_MAX_SIZE)

	n, _ := reader.Read(buf)
	fmt.Println(n)

	chunkType := buf[0] >> 6
	fmt.Printf("type %d\n", chunkType)

	var streamId uint32
	streamId = buf[0] & 0x3F
	fmt.Printf("stream %d\n", streamId)

	if streamId < 2 {
		streamId = buf[1] + 64
	} else if streamId >= 64 {
		streamId = 0
		streamId = uint32(buf[1])<<8 | buf[2]
	}
	return nil
}
