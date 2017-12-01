package rtmp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/golang/glog"
	"net"
)

const CHUNK_HEADER_MAX_SIZE = 18

type Chunk struct {
	Fmt               byte
	ChunkStreamId     uint32
	TimeStamp         uint32
	MessageLength     uint32
	MessageTypeId     byte
	MessageStreamId   uint32
	ExtendedTimestamp uint32
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

func readMessageHeader(reader *bufio.Reader, chunk *Chunk) error {
	var err error
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
		_, err = reader.Read(buf)
		if err != nil {
			return err
		}
		chunk.TimeStamp = binary.BigEndian.Uint32(buf[0:3])
		chunk.MessageLength = binary.BigEndian.Uint32(buf[3:6])
		chunk.MessageTypeId = buf[6:7][0]
		chunk.MessageStreamId = binary.BigEndian.Uint32(buf[7:11])
		glog.Info("Read message header type 0 done")

	} else if chunk.Fmt == 1 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id|
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		buf := make([]byte, 7, 7)
		_, err = reader.Read(buf)
		if err != nil {
			return err
		}
		chunk.TimeStamp = binary.BigEndian.Uint32(buf[0:3])
		chunk.MessageLength = binary.BigEndian.Uint32(buf[3:6])
		chunk.MessageTypeId = buf[6:7][0]
		glog.Info("Read message header type 1 done")
	} else if chunk.Fmt == 2 {
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		buf := make([]byte, 3, 3)
		_, err = reader.Read(buf)
		if err != nil {
			return err
		}
		chunk.TimeStamp = binary.BigEndian.Uint32(buf)
		glog.Info("Read message header type 2 done")
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
		_, err = reader.Read(buf)
		if err != nil {
			return err
		}
		chunk.ExtendedTimestamp = binary.BigEndian.Uint32(buf)
		glog.Info(fmt.Sprintf("Read extented timestamp: %v", chunk.ExtendedTimestamp))
	}
	return nil
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
