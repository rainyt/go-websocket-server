package net

import (
	"fmt"
	"net/http"
	"websocket_server/util"
)

// 管理服务器，实时监控每个服务器的状态
type ManagerServer struct {
	servers util.Array
}

// 开始侦听实时通用服务器的端口
func (m *ManagerServer) ListenServer(ip string, port int) {
	client := bind(ip, port)
	m.servers.Push(client)
}

// 侦听服务器
func (m *ManagerServer) Listen(port int) {
	http.HandleFunc("/get_server_port", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})
	http.ListenAndServe("0.0.0.0:"+fmt.Sprint(port), nil)
}
