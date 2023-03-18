package main

import (
	"flag"
	// "websocket_server/extends"
	"websocket_server/logs"
	"websocket_server/net"

	"go.uber.org/zap"
)

var (
	port  = flag.Int("port", 8888, "端口号")
	wss   = flag.Int("wss", 1, "是否开启wss，开启请填1，默认为0")
	ip    = flag.String("ip", "0.0.0.0", "ip地址")
	model = flag.String("model", "debug", "日志模式：debug/product")
)

func init() {
	flag.Parse()
	logs.InitLogger("./console.log", zap.DebugLevel, *model == "debug")
	logs.InfoF("启动参数, port = %d, ip = %s\n", *port, *ip)
}

func main() {
	s := net.Server{}
	// 初始化
	s.InitServer()
	// 注册V3的接口实现
	// s.Register(extends.V3Api{})
	if *wss == 1 {
		s.ListenTLS(*ip, *port)
	} else {
		s.Listen(*ip, *port)
	}
}
