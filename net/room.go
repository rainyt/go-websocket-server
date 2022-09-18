package net

import (
	"websocket_server/util"
)

type Room struct {
	id     int
	master *Client    // 房主
	users  util.Array // 房间用户
}

// 给房间的所有用户发送消息
func (r *Room) SendToAllUser(data []byte) {
	for _, v := range r.users.List {
		v.(Client).SendToUser(data)
	}
}

// 加入用户
func (r *Room) JoinClient(client *Client) {
	if client.room == nil {
		r.users.Push(client)
		client.room = r
	}
}

func (r *Room) ExitClient(client *Client) {
	if client.room != nil {
		if client.room.id == r.id {
			r.users.Remove(client)
			client.room = nil
		}
	}
}
