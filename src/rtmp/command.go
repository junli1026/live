package rtmp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/golang/glog"
	"net"
)

type Command struct {
	Name          string
	TransactionID uint32
}
