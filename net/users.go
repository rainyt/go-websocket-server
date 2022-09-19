package net

import "websocket_server/util"

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
	if u.users == nil {
		u.users = map[string]*RegisterUserData{}
	}
	user, err := u.users[openId]
	if err {
		// 用户曾经登陆过，需要检测用户是否在线，否则会发生挤出的事件
		if user.client != nil {
			util.Log("掉线处理")
			user.client.SendError(LOGIN_OUT_ERROR, "用户已在其他地方登录")
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
	return u.users[openId]
}
