package net

import (
	"sync"
	"websocket_server/util"
)

var lock sync.Mutex

type RegisterUserData struct {
	uid      int
	userName string
	client   *Client
}

// 用户数据数据库
type UserDataSQL struct {
	create_uid_index int
	users            map[string]*RegisterUserData
}

// 登陆角色
func (u *UserDataSQL) login(c *Client, openId string, userName string) *RegisterUserData {
	lock.Lock()
	defer lock.Unlock()
	user, err := u.users[openId]
	if err {
		// 用户曾经登陆过，需要检测用户是否在线，否则会发生挤出的事件
		if user.client != nil {
			if user.client.room != nil {
				// 如果原本就存在房间时，则需要把用户返回到房间中
				r := user.client.room
				r.ExitClient(user.client)
				r.JoinClient(c)
				util.Log("该用户仍然在房间中，加入房间")
			}
			util.Log("掉线处理")
			user.client.SendError(LOGIN_OUT_ERROR, Login, "用户已在其他地方登录")
			user.client.Close()
		}
	} else {
		// 新用户
		u.create_uid_index++
		u.users[openId] = &RegisterUserData{
			uid:      u.create_uid_index,
			userName: userName,
		}
		user = u.users[openId]
	}
	user.client = c
	c.uid = user.uid
	c.name = user.userName
	return u.users[openId]
}
