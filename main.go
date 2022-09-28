package main

import (
	"flag"
	"fmt"
	"websocket_server/net"
	"websocket_server/util"
)

var (
	port = flag.Int("port", 8888, "端口号")
	wss  = flag.Int("wss", 1, "是否开启wss，开启请填1，默认为0")
	ip   = flag.String("ip", "0.0.0.0", "ip地址")
)

func init() {
	flag.Parse()
}

func main() {
	util.EnableLog = true
	s := net.Server{}
	fmt.Printf("启动参数, port = %d, ip = %s\n", *port, *ip)
	if *wss == 1 {
		s.ListenTLS(*ip, *port)
	} else {
		s.Listen(*ip, *port)
	}
}
