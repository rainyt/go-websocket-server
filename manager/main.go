package main

import "websocket_server/manager/net"

var CurrentServer *net.ManagerServer

func main() {
	// 管理服的启动器
	CurrentServer = &net.ManagerServer{}
	ip := "192.168.0.110"
	CurrentServer.ListenServer(ip, 8888)
	CurrentServer.ListenServer(ip, 8889)
	CurrentServer.ListenServer(ip, 8890)
	CurrentServer.ListenServer(ip, 8891)
	CurrentServer.ListenServer(ip, 8892)
	CurrentServer.Listen(8080)
}
