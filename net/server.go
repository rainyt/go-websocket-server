package net

import (
	"fmt"
	"net"
	"websocket_server/util"
)

var CurrentServer *Server

type Server struct {
	users util.Array
}

func (s *Server) Listen(port int) {
	CurrentServer = s
	fmt.Println("Server start:127.0.0.1:" + fmt.Sprint(port))
	n, e := net.Listen("tcp", ":"+fmt.Sprint(port))
	if e != nil {
		fmt.Println(e.Error())
	}
	for {
		c, e := n.Accept()
		if e == nil {
			// 将用户写入到用户列表中
			s.users.Push(CreateClient(c))
		}
	}
}
