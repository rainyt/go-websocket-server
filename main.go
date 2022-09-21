package main

import (
	"fmt"
	"os"
	"strconv"
	"websocket_server/net"
	"websocket_server/util"
)

func main() {
	util.EnableLog = true
	s := net.Server{}
	fmt.Println("启动参数", os.Args)
	argsLen := len(os.Args)
	if argsLen == 2 {
		ip := "0.0.0.0"
		port, _ := strconv.Atoi(os.Args[1])
		s.Listen(ip, port)
	} else if argsLen == 3 {
		ip := os.Args[1]
		port, _ := strconv.Atoi(os.Args[2])
		s.Listen(ip, port)
	} else {
		panic("至少提供一个IP以及端口，参考命令：command 127.0.0.1 8888")
	}

}
