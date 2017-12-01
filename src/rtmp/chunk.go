package rtmp

import (
	"bufio"
	"github.com/golang/glog"
	"net"
)

const CHUNK_HEADER_MAX_SIZE = 18

type Chunk struct {
	Fmt           byte
	ChunkStreamId uint32
}

func readBasicHeader(reader *bufio.Reader, chunk *Chunk) error {
	b, err := reader.ReadByte()
	if err != nil {
		return err
	}

	chunk.Fmt = b >> 6
	streamId := uint32(b & 0x3F)

	if streamId < 2 {
		b, err = reader.ReadByte()
		chunk.ChunkStreamId = uint32(b) + 64
		if streamId == 1 {
			b, err = reader.ReadByte()
			chunk.ChunkStreamId += uint32(b) * 256
		}
	} else {
		chunk.ChunkStreamId = uint32(streamId)
	}

	return err
}

func readMessageHeader(reader *bufio, chunk *Chunk) error {
	if chunk.Fmt == 0 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                   timestamp                   |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id| msg stream id |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |           message stream id (cont)            |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

	} else if chunk.Fmt == 1 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id|
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

		_, err = ReadAtLeastFromNetwork(rbuf, tmpBuf[1:], 3)
		if err != nil {
			return
		}
		n += 3
		header.Timestamp = binary.BigEndian.Uint32(tmpBuf)
		_, err = ReadAtLeastFromNetwork(rbuf, tmpBuf[1:], 3)
		if err != nil {
			return
		}
		n += 3
		header.MessageLength = binary.BigEndian.Uint32(tmpBuf)
		b, err = ReadByteFromNetwork(rbuf)
		if err != nil {
			return
		}
		n += 1
		header.MessageTypeID = uint8(b)
	} else if chunk.Fmt == 2 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

	}
}

func ReadChunk(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	chunk := &Chunk{}
	var err error

	if err = readBasicHeader(reader, chunk); err != nil {
		goto exit
	}
	glog.Info("Read Chunk Basic Header done")

	if err = readMessageHeader(reader, chunk); err != nil {
		goto exit
	}
	glog.Info("Read Chunk Message Header done")

exit:
	glog.Error(err.Error())
	return err
}
