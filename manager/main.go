package main

import "websocket_server/manager/net"

var CurrentServer *net.ManagerServer

func main() {
	// 管理服的启动器
	CurrentServer = &net.ManagerServer{}
	CurrentServer.ListenServer("127.0.0.1", 8888)
	CurrentServer.ListenServer("127.0.0.1", 8889)
	CurrentServer.ListenServer("127.0.0.1", 8890)
	CurrentServer.ListenServer("127.0.0.1", 8891)
	CurrentServer.ListenServer("127.0.0.1", 8892)
	CurrentServer.Listen(8080)
}
