package websocketv2

import "websocket_server/logs"

type RegisterClient struct {
	// 客户端
	Client *WebSocket

	// 回调
	Callback chan int
}

// 服务器中心管理器
type ServerHub struct {
	// 绑定用户ID的客户端
	clientsByUserId map[string]*WebSocket

	// 客户端退出流程
	unregister chan *RegisterClient
}

// 创建服务器中心程序
func CreateServerHub() *ServerHub {
	return &ServerHub{
		clientsByUserId: make(map[string]*WebSocket),
		unregister:      make(chan *RegisterClient),
	}
}

var SERVER_HUB *ServerHub

// Init 启动服务器Hub事件循环，处理客户端注销（全局单例，仅允许初始化一次）
func Init() {
	if SERVER_HUB != nil {
		logs.ErrorF("ServerHub已初始化，忽略重复初始化")
		return
	}
	SERVER_HUB = CreateServerHub()
	for data := range SERVER_HUB.unregister {
		logs.InfoM("unregister", data.Client)
		if data.Client.OnUserOutCallback != nil {
			go data.Client.OnUserOutCallback()
		}
	}
}
