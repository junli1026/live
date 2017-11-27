package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/junli1026/live/src/rtmp"
	"net"
)

func init() {
	flag.Parse()
}

func handleConnection(conn net.Conn) {
	rtmp.HandShake(conn)
}

func main() {
	ln, err := net.Listen("tcp", ":1935")
	if err != nil {
		// handle error

	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			glog.Error("err")
		}
		go handleConnection(conn)
	}
	glog.Flush()
}
