package main

import (
	"flag"
	"log"
	"websocket_server/net"
	"websocket_server/util"
)

var (
	port      = flag.Int("port", 8888, "端口号")
	wss       = flag.Int("wss", 1, "是否开启wss，开启请填1，默认为0")
	ip        = flag.String("ip", "0.0.0.0", "ip地址")
	model     = flag.String("model", "debug", "日志模式：debug/product")
	enableLog = flag.Bool("log", false, "日志开关")
)

func init() {
	flag.Parse()
	util.InitLogger(model)
	util.EnableLog = *enableLog
	log.Printf("启动参数, port = %d, ip = %s\n", *port, *ip)
}

func main() {
	s := net.Server{}
	if *wss == 1 {
		s.ListenTLS(*ip, *port)
	} else {
		s.Listen(*ip, *port)
	}
}
