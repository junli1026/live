package rtmp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/golang/glog"
	"net"
)

const DEFAULT_CHUNK_SIZE = 128

type Chunk struct {
	Fmt               byte
	ChunkStreamId     uint32
	TimeStamp         uint32
	MessageLength     uint32
	MessageTypeId     byte
	MessageStreamId   uint32
	ExtendedTimestamp uint32
}

func readBasicHeader(reader *bufio.Reader, chunk *Chunk) (int, error) {
	var n int = 0
	b, err := reader.ReadByte()
	if err != nil {
		return n, err
	}
	n += 1

	chunk.Fmt = b >> 6
	streamId := uint32(b & 0x3F)

	if streamId < 2 {
		b, err = reader.ReadByte()
		if err != nil {
			n += 1
		}
		chunk.ChunkStreamId = uint32(b) + 64
		if streamId == 1 {
			b, err = reader.ReadByte()
			if err != nil {
				n += 1
			}
			chunk.ChunkStreamId += uint32(b) * 256
		}
	} else {
		chunk.ChunkStreamId = uint32(streamId)
	}

	return n, err
}

func readUint32(bytes []byte) uint32 {
	l := len(bytes)
	if l >= 4 {
		return binary.BigEndian.Uint32(bytes[0:4])
	} else {
		padding := make([]byte, 4-l, 4)
		return binary.BigEndian.Uint32(append(padding, bytes...))
	}
}

func dumpChunk(chunk *Chunk) string {
	return fmt.Sprintf("Chunk data: \n \tFmt: %d\n \tChunkStreamId: %d\n \tTimeStamp: %d\n \tMessageLength: %d\n \tMessageTypeId: %d\n \tMessageStreamId: %d\n \tExtendedTimestamp: %d\n",
		chunk.Fmt, chunk.ChunkStreamId, chunk.TimeStamp,
		chunk.MessageLength, chunk.MessageTypeId, chunk.MessageStreamId,
		chunk.ExtendedTimestamp)
}

func readMessageHeader(reader *bufio.Reader, chunk *Chunk) (int, error) {
	var err error = nil
	var n int = 0

	if chunk.Fmt == 0 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                   timestamp                   |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id| msg stream id |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |           message stream id (cont)            |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		buf := make([]byte, 11, 11)
		n, err = reader.Read(buf)
		if err != nil {
			return n, err
		}
		chunk.TimeStamp = readUint32(buf[0:3])
		chunk.MessageLength = readUint32(buf[3:6])
		chunk.MessageTypeId = buf[6:7][0]
		chunk.MessageStreamId = readUint32(buf[7:11])
	} else if chunk.Fmt == 1 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id|
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		buf := make([]byte, 7, 7)
		n, err = reader.Read(buf)
		if err != nil {
			return n, err
		}
		chunk.TimeStamp = readUint32(buf[0:3])
		chunk.MessageLength = readUint32(buf[3:6])
		chunk.MessageTypeId = buf[6:7][0]
		glog.Info("Read message header type 1 done")
	} else if chunk.Fmt == 2 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		buf := make([]byte, 3, 3)
		n, err = reader.Read(buf)
		if err != nil {
			return n, err
		}
		chunk.TimeStamp = readUint32(buf)
	}

	// The Extended Timestamp field is used to encode timestamps or
	// timestamp deltas that are greater than 16777215 (0xFFFFFF); that is,
	// for timestamps or timestamp deltas that don’t fit in the 24 bit
	// fields of Type 0, 1, or 2 chunks. This field encodes the complete
	// 32-bit timestamp or timestamp delta. The presence of this field is
	// indicated by setting the timestamp field of a Type 0 chunk, or the
	// timestamp delta field of a Type 1 or 2 chunk, to 16777215 (0xFFFFFF).
	// This field is present in Type 3 chunks when the most recent Type 0,
	// 1, or 2 chunk for the same chunk stream ID indicated the presence of
	// an extended timestamp field.

	// TODO: handle the following case:
	//    "This field is present in Type 3 chunks when the most recent Type 0,
	//    1, or 2 chunk for the same chunk stream ID indicated the presence of
	//    an extended timestamp field."
	if chunk.Fmt != 3 && chunk.TimeStamp >= 0xFFFFFF {
		buf := make([]byte, 4, 4)
		var c int
		c, err = reader.Read(buf)
		if err != nil {
			return n, err
		}
		n += c
		chunk.ExtendedTimestamp = readUint32(buf)
	}
	fmt.Println(dumpChunk(chunk))
	glog.Info(dumpChunk(chunk))
	return n, nil
}

func ReadChunk(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	buf := make([]byte, 1024)
	chunk := &Chunk{}
	var chunkSize uint32 = DEFAULT_CHUNK_SIZE
	var err error = nil
	var n, c int = 0, 0

	// read basic header
	if c, err = readBasicHeader(reader, chunk); err != nil {
		goto exit
	}
	n += c
	glog.Info("Read Chunk Basic Header done")

	// read message header
	if c, err = readMessageHeader(reader, chunk); err != nil {
		goto exit
	}
	n += c
	glog.Info("Read Chunk Message Header done")

	fmt.Println(fmt.Sprintf("chunk header size: %v", n))

	// read chunk data
	if chunk.MessageLength < chunkSize {
		chunkSize = chunk.MessageLength
	}

	fmt.Println(fmt.Sprintf("chunk data size: %v", chunkSize))

	c, err = reader.Read(buf[0 : chunkSize-uint32(n)])
	fmt.Println(c)

	return nil
exit:
	glog.Error(err.Error())
	return err
}
