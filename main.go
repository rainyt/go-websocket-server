package main

import (
	"websocket_server/net"
	"websocket_server/util"
)

func main() {
	util.EnableLog = true
	s := net.Server{}
	s.Listen(8888)
}
