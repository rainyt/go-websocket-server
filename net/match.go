package net

import "websocket_server/util"

// 匹配管理
type Matchs struct {
	matchUsers *util.Array // 匹配用户
}

// 匹配用户
func (m *Matchs) matchUser(c *Client) bool {
	if m.matchUsers.IndexOf(c) == -1 {
		m.matchUsers.Push(c)
		// 开始匹配算法
		m.mathMatch()
		return true
	}
	return false
}

// 取消匹配
func (m *Matchs) cannelMatchUser(c *Client) bool {
	return m.matchUsers.Remove(c)
}

// 计算匹配结果，如果有符合的，则开始处理
func (m *Matchs) mathMatch() {

}
