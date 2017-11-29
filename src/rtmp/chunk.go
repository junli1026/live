package rtmp

import (
	"bufio"
	"github.com/golang/glog"
	"net"
)

const CHUNK_HEADER_MAX_SIZE = 18

type Chunk struct {
	Type          byte
	ChunkStreamId uint32
}

func readBasicHeader(reader *bufio.Reader, chunk *Chunk) error {
	b, err := reader.ReadByte()
	if err != nil {
		return err
	}

	chunk.Type = b >> 6
	chunk.ChunkStreamId = uint32(b & 0x3F)

	if chunk.ChunkStreamId < 2 {
		b, err = reader.ReadByte()
		chunk.ChunkStreamId += uint32(b) + 64
	}
	if chunk.ChunkStreamId == 1 {
		b, err = reader.ReadByte()
		chunk.ChunkStreamId += uint32(b) * 256
	}
	return err
}

func readMessageHeader(reader *bufio, chunk *Chunk) error {
	//TODO
	return nil
}

func ReadChunk(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	chunk := &Chunk{}
	var err error

	if err = readBasicHeader(reader, chunk); err != nil {
		goto exit
	}
	if err = readMessageHeader(reader, chunk); err != nil {
		goto exit
	}

exit:
	glog.Error(err.Error())
	return err
}
