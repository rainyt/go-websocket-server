package websocketv2

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

	// 客户端注册流程
	register chan *RegisterClient

	// 客户端退出流程
	unregister chan *RegisterClient

	// 广播消息
	// broadcast chan *response.ClientJsonMessage

	// 指定广播
	// target_broadcast chan *response.ClientJsonMessage

	// 使服务器进入维护
	// server_uphold chan *response.ClientJsonMessage
}

// 创建服务器中心程序
func CreateServerHub() *ServerHub {
	return &ServerHub{
		clientsByUserId: make(map[string]*WebSocket),
		register:        make(chan *RegisterClient),
		unregister:      make(chan *RegisterClient),
		// broadcast:        make(chan *response.ClientJsonMessage),
		// target_broadcast: make(chan *response.ClientJsonMessage),
		// server_uphold:    make(chan *response.ClientJsonMessage),
	}
}

var SERVER_HUB *ServerHub

func Init() {
	SERVER_HUB = CreateServerHub()
	for {
		select {
		case data := <-SERVER_HUB.unregister:
			data.Client.OnUserOutCallback()
		}
	}
}
