package net

import (
	"fmt"
	"sync"
	"time"
	"websocket_server/logs"
	"websocket_server/runtime"
	"websocket_server/util"
)

// 客户端的状态同步使用的数据结构
type ClientState struct {
	Data *util.Map `json:"data"` // 客户端状态同步的所用到的数据储存在这里
}

// 房间可选参数
type RoomConfigOption struct {
	maxCounts int     // 房间最大容纳人数
	password  string  // 房间密码，加入房间时，需要验证密码
	fps       float64 // 帧同步帧率，0 表示使用默认值 30
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
	removed       bool                 // 是否已从 App 中移除（防止重复回收房间ID）
	removeMu      sync.Mutex           // 保护 removed 标志的并发安全
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
	if r == nil || r.users == nil || r.users.List == nil {
		return true
	}
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
	defer runtime.GoRecover()
	for {
		// app := r.master.getApp()
		if !r.frameSync || r.isInvalidRoom() {
			// 帧同步停止，或者房间已经不存在用户时
			logs.InfoM("房间停止帧同步")
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
				if v2 != nil {
					f, b := v2.(FrameData)
					if b {
						a = append(a, f.Data)
					}
				}
			}
			if a != nil {
				frameData[c.uid] = a
			}
			c.frames.List = []any{}
		}

		// 缓存数据
		r.cacheId++
		r.frameDatas.Push(frameData)
		// 发送帧数据到客户端（frameData 直接传入，由 SendToUserOp 内部 Marshal）
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
	logs.InfoM("StartFrameSync")
	r.frameSync = true
	r.lock = true
	// 所有人都要接收这个字节，确保帧同步启动
	r.SendToAllUserOp(&ClientMessage{
		Op: FrameSyncReady,
	}, nil)
	go onRoomFrame(r)
	// 通知大厅房间列表变更
	r.master.getApp().broadcastRoomListChanged()
}

// 停止帧同步
func (r *Room) StopFrameSync(keepLock bool) {
	r.frameSync = false
	if !keepLock {
		r.lock = false
	}
	r.cacheId = 0
	// 清理房间的僵尸玩家（离线但未退出房间的玩家）
	r.cleanZombieClients()
	// 通知大厅房间列表变更
	r.master.getApp().broadcastRoomListChanged()
	logs.InfoM("StopFrameSync")
}

// 清理房间的僵尸玩家（离线但未退出房间的玩家）
func (r *Room) cleanZombieClients() {
	var zombies []*Client
	for _, v := range r.users.List {
		c := v.(*Client)
		if !c.Connected {
			zombies = append(zombies, c)
		}
	}
	for _, c := range zombies {
		logs.InfoM("清理僵尸玩家:", c.name, "房间ID:", r.id)
		r.ExitClient(c)
	}
}

// 自动分配最小可用座位号
func (r *Room) assignSeat(client *Client) {
	taken := map[int]bool{}
	for _, v := range r.users.List {
		c := v.(*Client)
		if c != client && c.seat > 0 {
			taken[c.seat] = true
		}
	}
	for seat := 1; seat <= r.option.maxCounts; seat++ {
		if !taken[seat] {
			client.seat = seat
			logs.InfoM(client.name, "自动分配座位:", seat)
			return
		}
	}
}

// 更换座位
func (r *Room) SwitchSeat(client *Client, targetSeat int) error {
	if targetSeat < 1 || targetSeat > r.option.maxCounts {
		return fmt.Errorf("座位号无效，有效范围为1~%d", r.option.maxCounts)
	}
	// 检查目标座位是否已被占用
	for _, v := range r.users.List {
		c := v.(*Client)
		if c != client && c.seat == targetSeat {
			return fmt.Errorf("座位%d已被占用", targetSeat)
		}
	}
	oldSeat := client.seat
	client.seat = targetSeat
	logs.InfoM(client.name, "更换座位:", oldSeat, "->", targetSeat, "房间ID:", r.id)
	// 通知房间内所有成员座位变更
	r.SendToAllUserOp(&ClientMessage{
		Op: SeatUpdate,
		Data: map[string]any{
			"uid":     client.uid,
			"oldSeat": oldSeat,
			"newSeat": targetSeat,
		},
	}, nil)
	// 通知房间信息更新
	r.onRoomChanged()
	return nil
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
		// 自动分配最小可用座位号
		r.assignSeat(client)
		logs.InfoM(client.name, "加入房间["+fmt.Sprint(r.id)+"]，当前房间人数：", r.users.Length())
		client.room = r
		logs.InfoM("发送房间消息给用户", client.name)
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
		// 通知大厅房间列表变更
		client.getApp().broadcastRoomListChanged()
		logs.InfoM("加入用户行为结束", client.name)
	}
}

func (r *Room) onRoomChanged() {
	r.SendToAllUserOp(&ClientMessage{
		Op: ChangedRoom,
	}, nil)
}

// 用户退出
func (r *Room) ExitClient(client *Client) {
	if client.room != nil && r.users != nil && r.users.List != nil {
		if client.room.id == r.id {
			// 记录退出前的座位号，用于判断是否需要取消角色选择
			exitedSeat := client.seat

			r.users.Remove(client)
			client.room = nil
			client.seat = 0
			if r.users.Length() == 0 {
				// 房间已经不存在用户了，则删除当前房间
				client.getApp().removeRoom(r)
			} else {
				// 如果用户仍然存在时，如果是房主掉线，则需要更换房主。不管房主是否更换，都需要通知客户端用户重新更新房间信息
				if r.master == client {
					r.master = r.users.List[0].(*Client)
				}
				// 需要将状态清空（最小化锁范围，仅锁住 map 写操作，避免持有锁期间发送消息导致死锁）
				r.userStateLock.Lock()
				r.userState[client.uid] = nil
				r.userStateLock.Unlock()

				// 同步退出用户信息
				r.SendToAllUserOp(&ClientMessage{
					Op:   ExitRoomClient,
					Data: client.GetUserData(),
				}, client)

				// 如果退出的是参战方（座位1或2），通知所有成员取消角色选择
				if exitedSeat == 1 || exitedSeat == 2 {
					r.SendToAllUserOp(&ClientMessage{
						Op: RoomMessage,
						Data: map[string]any{
							"uid": client.uid,
							"data": map[string]any{
								"type": "cancelRoleSelect",
							},
						},
					}, nil)
				}

				// 通知更新房间信息
				r.onRoomChanged()
				// 通知大厅房间列表变更
				client.getApp().broadcastRoomListChanged()

				logs.InfoM(client.name, "：离开房间["+fmt.Sprint(r.id)+"]，当前房间人数：", r.users.Length())
			}
		}
	} else {
		client.getApp().removeRoom(r)
	}
}

// 获取房间信息
func (r *Room) GetRoomData() any {
	data := map[string]any{}
	data["id"] = r.id
	data["master"] = r.master.GetUserData()
	// 用户数据
	users := util.Array{}
	seats := map[int]int{} // 座位映射表：{座位号: uid}
	for _, v := range r.users.List {
		c := v.(*Client)
		users.Push(c.GetUserData())
		if c.seat > 0 {
			seats[c.seat] = c.uid
		}
	}
	data["users"] = users.List
	data["seats"] = seats
	data["max"] = r.option.maxCounts
	data["data"] = r.customData.Copy()
	data["state"] = r.roomState.Data.Copy()
	var state map[int]any = map[int]any{}
	r.userStateLock.Lock()
	defer r.userStateLock.Unlock()
	for k, cs := range r.userState {
		if cs != nil {
			state[k] = cs.Data.Copy()
		}
	}
	data["usersState"] = state
	return data
}
