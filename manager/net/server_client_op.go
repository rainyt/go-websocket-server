package net

import (
	"fmt"
	"strings"
	"websocket_server/util"
	"websocket_server/websocket"
)

// 消息处理
func (s *ServerClient) onMessage() {
	if s.state == websocket.Handshake {
		// 握手实现
		index := strings.Index(string(s.data.Data), "\r\n\r\n")
		if index != -1 {
			// 接收到完整的握手信息
			// fmt.Println(string(s.data.Data))
			// todo 这里需要做连接验证
			s.state = websocket.Head
			util.Log(s.ip + ":" + fmt.Sprint(s.port) + " connected")
		}
	} else {
		// 读取websocket交互包
	}
}
