package net

import "websocket_server/util"

// 匹配管理
type Matchs struct {
	matchUsers *util.Array // 匹配用户
}

// 房间、玩家之间匹配可选参数
type MatchOption struct {
	key string         // 匹配key，字符串比较，当为一样的时候，则对匹配，如果为空字符串时，则忽略此匹配
	min map[string]int // 匹配参数的最小值
	max map[string]int // 匹配参数的最大值
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
