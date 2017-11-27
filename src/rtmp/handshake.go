package rtmp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"math/rand"
	"net"
	"time"
)

const RTMP_SIG_SIZE = 1536

func timeStamp() []byte {
	t := time.Now().Unix()
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(t))
	return b[4:]
}

func HandShake(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	var err error
	var n int
	var c1 []byte
	c0c1 := make([]byte, RTMP_SIG_SIZE+1, RTMP_SIG_SIZE+1)
	c2 := make([]byte, RTMP_SIG_SIZE, RTMP_SIG_SIZE)
	s0 := []byte{3}
	s1 := make([]byte, RTMP_SIG_SIZE, RTMP_SIG_SIZE)
	//uptime := make([]byte, 4, 4)
	zeros := make([]byte, 4, 4)

	n, err = reader.Read(c0c1)
	if err != nil || (n != 1 && n != RTMP_SIG_SIZE+1) {
		glog.Error("HandShake: read c0c1 failed.")
		goto exit
	}

	glog.Info("c0: %d\n", c0c1[0])
	if c0c1[0] != 3 {
		err = errors.New("HandShake: c0 is not Ox03")
		goto exit
	}

	_, err = writer.Write(s0)
	if err != nil {
		goto exit
	}

	if n == 1 {
		c1 = make([]byte, RTMP_SIG_SIZE, RTMP_SIG_SIZE)
		n, err = reader.Read(c1)
		if err != nil {
			glog.Error("HandShake: read c1 failed.")
			goto exit
		}
	} else {
		c1 = c0c1[1:]
	}
	//uptime = c1[0:4]
	zeros = c1[4:8]
	if zeros[0] != 0 || zeros[1] != 0 || zeros[2] != 0 || zeros[3] != 0 {
		err = errors.New(fmt.Sprintf("c1 zeros is invald: %v\n", zeros))
		goto exit
	}

	rand.Read(s1)
	copy(s1[0:4], timeStamp()[:])
	copy(s1[4:8], []byte{0, 0, 0, 0})
	_, err = writer.Write(s1)
	if err != nil {
		goto exit
	}

	// s2 is same as c1
	_, err = writer.Write(c1)
	if err != nil {
		goto exit
	}
	writer.Flush()

	n, err = reader.Read(c2)
	if err != nil {
		goto exit
	}

	glog.Info(fmt.Sprintf("Received c2, %d bytes\n", n))
	glog.Info("HandShake succeed")
	return nil
exit:
	glog.Error(err.Error())
	return err
}
