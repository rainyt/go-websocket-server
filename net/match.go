package net

import (
	"websocket_server/logs"
	"websocket_server/util"
)

// 匹配组
type MatchGroup struct {
	users  *util.Array  // 当前已经匹配的用户
	option *MatchOption // 匹配参数值
}

// 匹配管理
type Matchs struct {
	matchUsers      *util.Array // 匹配用户
	matchUsersGroup *util.Array // 匹配用户组
}

// 匹配的范围值
type MatchRange struct {
	Min int `json:"min"` // 最小值
	Max int `json:"max"` // 最大值
}

// 房间、玩家之间匹配可选参数，匹配参数会跟用户的data参数进行匹配
type MatchOption struct {
	Key    string                `json:"key"`    // 匹配key，字符串比较，当为一样的时候，则对匹配，如果为空字符串时，则忽略此匹配
	Number int                   `json:"number"` // 匹配所需的总人数
	Range  map[string]MatchRange `json:"range"`  // 匹配参数的最小值，到最大值
}

// 匹配用户
func (m *Matchs) matchUser(c *Client, option *MatchOption) bool {
	if m.matchUsers.IndexOf(c) == -1 {
		m.matchUsers.Push(c)
		c.matchOption = option
		// 开始匹配算法
		m.mathMatch(c)
		return true
	}
	return false
}

// 取消匹配，需要从匹配组移除
func (m *Matchs) cannelMatchUser(c *Client) bool {
	c.matchOption = nil
	for _, v := range m.matchUsersGroup.List {
		mg := v.(*MatchGroup)
		if mg.users.IndexOf(c) != -1 {
			mg.users.Remove(c)
			if mg.users.Length() == 0 {
				m.matchUsersGroup.Remove(mg)
			}
			break
		}
	}
	return m.matchUsers.Remove(c)
}

// 计算匹配结果，如果有符合的，则开始处理
func (m *Matchs) mathMatch(c *Client) {
	// 算法，匹配时从UsersGroup中最后一个用户进行匹配
	for _, v := range m.matchUsersGroup.List {
		mg := v.(*MatchGroup)
		if mg.option.matchClient(c) {
			mg.users.Push(c)
			// 符合匹配人数时，则开始创建一个新的房间
			if mg.option.Number == mg.users.Length() {
				logs.InfoM("匹配成功")
				// 创建一个匹配好的房间
				room := c.getApp().CreateRoom(mg.users.List[0].(*Client), RoomConfigOption{
					maxCounts: mg.option.Number,
					password:  "",
				})
				if room == nil {
					logs.InfoM("匹配错误，房间不存在")
					return
				}
				otherClients := mg.users.List[1:]
				for _, v2 := range otherClients {
					room.JoinClient(v2.(*Client))
				}
				// 并通知所有玩家匹配成功
				room.SendToAllUserOp(&ClientMessage{Op: Matched}, nil)
				// 所有玩家退出匹配
				for _, v3 := range mg.users.List {
					m.cannelMatchUser(v3.(*Client))
				}
			}
			return
		}
	}
	// 没有成功匹配到任何一个组，则需要创建一个新的用户组
	group := &MatchGroup{
		users:  util.CreateArray(),
		option: c.matchOption,
	}
	group.users.Push(c)
	m.matchUsersGroup.Push(group)
}

// 匹配客户端用户
func (o *MatchOption) matchClient(c *Client) bool {
	logs.InfoM("[matchClient]")
	if c.matchOption == nil {
		logs.InfoM("用户没有匹配参数")
		return false
	}
	// 需先验证Key是否一致，或者无要求
	if o.Key == c.matchOption.Key {
		// 人数验证
		if o.Number == c.matchOption.Number {
			// 参数验证
			for k, mr := range o.Range {
				v := c.userData.GetData(k, nil)
				if v != nil {
					i, b2 := v.(int)
					if b2 {
						if !(mr.Min <= i && i <= mr.Max) {
							logs.InfoM("匹配验证错误：范围值不匹配")
							return false
						}
					} else {
						logs.InfoM("匹配验证错误：无法获取" + k + "比较参数")
						return false
					}
				} else {
					logs.InfoM("匹配验证错误：range不匹配")
					return false
				}
			}
		} else {
			logs.InfoM("匹配验证错误：房间人数不匹配")
			return false
		}
	} else {
		logs.InfoM("匹配验证错误：key不匹配")
		return false
	}
	return true
}
