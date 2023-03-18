package net

import (
	"sync"
	"websocket_server/logs"
)

// 注册的用户数据
type RegisterUserData struct {
	uid        int     // 用户唯一ID
	userName   string  // 用户昵称
	client     *Client // 用户引用的客户端
	CustomData any     // 自定义数据，该参数在扩展自定义服务API时，可直接使用
}

// 用户数据数据库
type UserDataSQL struct {
	create_uid_index int                          // UID创建索引
	lock             sync.Mutex                   // 创建锁
	users            map[string]*RegisterUserData // 注册数据
}

// 登陆角色
func (u *UserDataSQL) login(c *Client, openId string, userName string) *RegisterUserData {
	u.lock.Lock()
	user, err := u.users[openId]
	if err {
		// 用户曾经登陆过，需要检测用户是否在线，否则会发生挤出的事件
		if user.client != nil {
			if user.client.room != nil {
				// 如果原本就存在房间时，则需要把用户返回到房间中
				r := user.client.room
				r.ExitClient(user.client)
				r.JoinClient(c)
				logs.InfoM("该用户[" + user.client.name + "]仍然在房间中，加入房间")
			}
			logs.InfoM(user.client.name + "掉线处理")
			user.client.SendError(LOGIN_OUT_ERROR, Login, "用户已在其他地方登录")
			user.client.Close()
			user.userName = userName
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
	u.lock.Unlock()
	return u.users[openId]
}

// 通过UID获取用户数据
func (u *UserDataSQL) GetUserDataByUid(uid int) *RegisterUserData {
	u.lock.Lock()
	defer u.lock.Unlock()
	for _, rud := range u.users {
		if rud.uid == uid {
			if rud.client != nil {
				return rud
			}
		}
	}
	return nil
}
