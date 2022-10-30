package net

import (
	"fmt"
	"sync"
	"time"
	"websocket_server/util"
)

// 客户端的状态同步使用的数据结构
type ClientState struct {
	Data *util.Map `json:"data"` // 客户端状态同步的所用到的数据储存在这里
}

// 房间可选参数
type RoomConfigOption struct {
	maxCounts int    // 房间最大容纳人数
	password  string // 房间密码，加入房间时，需要验证密码
}

type Room struct {
	id            int
	master        *Client              // 房主
	users         *util.Array          // 房间用户
	roomState     *ClientState         // 房间端的状态栏同步（每个用户都可以共享修改的内容）
	userStateLock sync.Mutex           // 锁定
	userState     map[int]*ClientState // 客户端状态数据同步
	frameSync     bool                 // 是否开启帧同步
	interval      time.Duration        // 帧同步的间隔
	lock          bool                 // 房间是否锁定（如果游戏已经开始，则会锁定房间，直到游戏结束，如果用户离线，不会立即退出房间，需要通过`ExitRoom`才能退出房间）
	frameDatas    *util.Array          // 房间帧数据
	cacheId       int                  // 房间已缓存的时间轴Id
	option        *RoomConfigOption    // 房间可选参数
	matchOption   *MatchOption         // 房间匹配参数
	customData    *util.Map            // 房间自定义数据
	oldMsgs       *util.Array          // 历史消息，会记录所有`RoomMessage`信息
}

// 更新自定义数据
func (r *Room) updateCustomData(o any) {
	obj, bool := o.(map[string]any)
	if bool {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		for _, v := range keys {
			r.customData.Store(v, obj[v])
		}
	}
}

// 更新房间配置
func (r *Room) updateRoomData(data RoomConfigOption) {
	// 当放人数大于最大人数时，则以最大人数来处理
	if r.users.Length() > data.maxCounts {
		data.maxCounts = r.users.Length()
	}
	r.option = &data
}

// 是否为无效房间
func (r *Room) isInvalidRoom() bool {
	hasOnline := false
	for _, v := range r.users.List {
		if v.(*Client).Connected {
			hasOnline = true
		}
	}
	return r.users.Length() == 0 || !hasOnline
}

// 将玩家踢出房间
func (r *Room) kickOut(uid int) {
	for _, v := range r.users.List {
		c := v.(*Client)
		if c.uid == uid {
			r.ExitClient(c)
			// 发送踢出房间的事件
			c.SendToUserOp(&ClientMessage{
				Op: SelfKickOut,
			})
			r.onRoomChanged()
			break
		}
	}
}

// 房间的帧同步实现
func onRoomFrame(r *Room) {
	for {
		// app := r.master.getApp()
		if !r.frameSync || r.isInvalidRoom() {
			// 帧同步停止，或者房间已经不存在用户时
			util.Log("房间停止帧同步")
			// 并将所有用户移除
			// for _, v := range r.users.List {
			// app.ExitRoom(v.(*Client))
			// }
			break
		}
		frameData := map[int][]any{}
		// 收集房间的所有用户操作
		for _, v := range r.users.List {
			c := v.(*Client)
			a := frameData[c.uid]
			for _, v2 := range c.frames.List {
				f := v2.(FrameData)
				a = append(a, f.Data)
			}
			if a != nil {
				frameData[c.uid] = a
			}
			c.frames.List = []any{}
		}

		// 缓存数据
		r.cacheId++
		r.frameDatas.Push(frameData)

		// 发送帧数据到客户端
		for _, v := range r.users.List {
			c := v.(*Client)
			c.SendToUserOp(&ClientMessage{
				Op: FData,
				Data: map[string]any{
					"t": r.cacheId,
					"d": frameData,
				},
			})
		}
		// 帧同步发送间隔
		time.Sleep(r.interval)
	}
}

// 记录服务器的房间信息
func (r *Room) recordRoomMessage(data *ClientMessage) {
	r.oldMsgs.Push(data)
	// 超出时，把第一个删除
	if r.oldMsgs.Length() > 200 {
		r.oldMsgs.Remove(r.oldMsgs.List[0])
	}
}

// 启动帧同步
func (r *Room) StartFrameSync() {
	if r.frameSync {
		return
	}
	util.Log("StartFrameSync")
	r.frameSync = true
	r.lock = true
	// 所有人都要接收这个字节，确保帧同步启动
	r.SendToAllUserOp(&ClientMessage{
		Op: FrameSyncReady,
	}, nil)
	go onRoomFrame(r)
}

// 停止帧同步
func (r *Room) StopFrameSync() {
	r.frameSync = false
	r.lock = false
	r.cacheId = 0
	util.Log("StopFrameSync")
}

// 给房间的所有用户发送消息
func (r *Room) SendToAllUser(data []byte) {
	for _, v := range r.users.List {
		v.(*Client).SendToUser(data)
	}
}

// 给房间的所有用户发送消息
func (r *Room) SendToAllUserOp(data *ClientMessage, igoneClient *Client) {
	for _, v := range r.users.List {
		if v != igoneClient {
			v.(*Client).SendToUserOp(data)
		}
	}
}

// 加入用户
func (r *Room) JoinClient(client *Client) {
	if client.room == nil {
		r.users.Push(client)
		util.Log(client.name, "加入房间["+fmt.Sprint(r.id)+"]，当前房间人数：", r.users.Length())
		client.room = r
		client.SendToUserOp(&ClientMessage{
			Op:   GetRoomData,
			Data: r.GetRoomData(),
		})
		// 同步新来用户信息
		r.SendToAllUserOp(&ClientMessage{
			Op:   JoinRoomClient,
			Data: client.GetUserData(),
		}, client)
		// 其他用户通知房间更新
		r.onRoomChanged()
	}
}

func (r *Room) onRoomChanged() {
	r.SendToAllUserOp(&ClientMessage{
		Op: ChangedRoom,
	}, nil)
}

// 用户退出
func (r *Room) ExitClient(client *Client) {
	if client.room != nil {
		if client.room.id == r.id {
			r.users.Remove(client)
			client.room = nil
			if r.users.Length() == 0 {
				// 房间已经不存在用户了，则删除当前房间
				client.getApp().rooms.Remove(r)
			} else {
				// 如果用户仍然存在时，如果是房主掉线，则需要更换房主。不管房主是否更换，都需要通知客户端用户重新更新房间信息
				if r.master == client {
					r.master = r.users.List[0].(*Client)
				}
				// 需要将状态清空
				r.userStateLock.Lock()
				r.userState[client.uid] = nil
				r.userStateLock.Unlock()
				// 同步退出用户信息
				r.SendToAllUserOp(&ClientMessage{
					Op:   ExitRoomClient,
					Data: client.GetUserData(),
				}, client)
				// 通知更新房间信息
				r.onRoomChanged()

				util.Log(client.name, "：离开房间["+fmt.Sprint(r.id)+"]，当前房间人数：", r.users.Length())
			}
		}
	}
}

// 获取房间信息
func (r *Room) GetRoomData() any {
	data := map[string]any{}
	data["id"] = r.id
	data["master"] = r.master.GetUserData()
	// 用户数据
	users := util.Array{}
	for _, v := range r.users.List {
		users.Push(v.(*Client).GetUserData())
	}
	data["users"] = users.List
	data["max"] = r.option.maxCounts
	data["data"] = r.customData.Data
	data["state"] = r.roomState.Data.Data
	var state map[int]any = map[int]any{}
	for k, cs := range r.userState {
		if cs != nil {
			state[k] = cs.Data.Data
		}
	}
	data["usersState"] = state
	return data
}
