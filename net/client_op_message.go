package net

import (
	"encoding/json"
	"fmt"
	"websocket_server/util"
)

// 消息处理
func (c *Client) onMessage(data []byte) {
	// 解析API操作
	message := ClientMessage{}
	var err error
	// 如果是二进制数据，则需要解析处理，第一位是op操作符，剩余的是内容
	if c.frameIsBinary {
		op := ClientAction(data[0])
		content := data[1:]
		message.Op = op
		err = json.Unmarshal(content, &message.Data)
	} else {
		err = json.Unmarshal(data, &message)
	}
	if err == nil {
		fmt.Println("处理命令", message)
		if c.uid == 0 || message.Op == Login {
			switch message.Op {
			case Login:
				if c.uid == 0 {
					loginData := message.Data.(map[string]any)
					userName := loginData["username"]
					openId := loginData["openid"]
					if userName == nil || openId == nil {
						c.SendError(LOGIN_ERROR, message.Op, "需要提供openid和username")
						return
					}
					// 只需要用户名和OpenId即可登陆
					userData := CurrentServer.usersSQL.login(c, openId.(string), userName.(string))
					util.Log("登陆成功：", userData)
					c.SendToUserOp(&ClientMessage{
						Op: Login,
						Data: map[string]any{
							"uid": userData.uid,
						},
					})
				} else {
					c.SendToUserOp(&ClientMessage{
						Op: Login,
						Data: map[string]any{
							"uid": c.uid,
						},
					})
					// c.SendError(LOGIN_ERROR, message.Op, "已登陆")
				}
			default:
				c.SendError(OP_ERROR, message.Op, "无效的操作指令："+fmt.Sprint(message.Op))
			}
			return
		}
		switch message.Op {
		case Message:
			// 接收到消息
			fmt.Println("服务器接收到消息：", message.Data)
		case CreateRoom:
			// 创建一个房间
			room := CurrentServer.CreateRoom(c, RoomConfigOption{})
			util.Log("开始创建房间", room)
			if room != nil {
				// 创建成功
				c.SendToUserOp(&ClientMessage{
					Op: CreateRoom,
					Data: map[string]any{
						"id": room.id,
					},
				})
			} else {
				// 创建失败
				c.SendError(CREATE_ROOM_ERROR, message.Op, "房间已存在，无法创建")
			}
		case GetRoomData:
			// 获取房间信息
			if c.room != nil {
				c.SendToUserOp(&ClientMessage{
					Op:   GetRoomData,
					Data: c.room.GetRoomData(),
				})
			} else {
				c.SendError(GET_ROOM_ERROR, message.Op, "不存在房间信息")
			}
		case JoinRoom:
			if c.room != nil {
				c.SendError(JOIN_ROOM_ERROR, message.Op, "已存在房间，无法加入")
			} else {
				id := util.GetMapValueToInt(message.Data, "id")
				password := util.GetMapValueToString(message.Data, "password")
				room, b := CurrentServer.JoinRoom(c, id, password)
				if b {
					c.SendToUserOp(&ClientMessage{
						Op: JoinRoom,
						Data: map[string]any{
							"id": room.id,
						}},
					)
				} else {
					c.SendError(JOIN_ROOM_ERROR, message.Op, "无法加入该房间")
				}
			}
		case ExitRoom:
			if c.room != nil {
				c.room.ExitClient(c)
				c.SendToUserOp(&ClientMessage{
					Op: ExitRoom,
				})
			} else {
				c.SendError(EXIT_ROOM_ERROR, message.Op, "退出房间失败")
			}
		case StartFrameSync:
			// 开始帧同步
			if c.room != nil {
				c.room.StartFrameSync()
				c.SendToUserOp(&ClientMessage{
					Op: StartFrameSync,
				})
			} else {
				c.SendError(START_FRAME_SYNC_ERROR, message.Op, "房间不存在，无法启动帧同步")
			}
		case StopFrameSync:
			// 开始停止帧同步
			if c.room != nil {
				c.room.StopFrameSync()
				c.SendToUserOp(&ClientMessage{
					Op: StopFrameSync,
				})
			} else {
				c.SendError(STOP_FRAME_SYNC_ERROR, message.Op, "房间不存在，无法停止帧同步")
			}
		case UploadFrame:
			if c.room != nil && c.room.frameSync {
				// 缓存到用户数据中
				fdata := FrameData{
					Time: 0,
					Data: message.Data,
				}
				c.frames.Push(fdata)
				c.SendToUserOp(&ClientMessage{
					Op: UploadFrame,
				})
			} else {
				c.SendError(UPLOAD_FRAME_ERROR, message.Op, "上传帧同步数据错误")
			}
		case RoomMessage:
			// 转发房间信息
			if c.room != nil {
				c.room.recordRoomMessage(message)
				c.room.SendToAllUserOp(&message, c)
				c.SendToUserOp(&ClientMessage{
					Op: RoomMessage,
				})
			} else {
				c.SendError(SEND_ROOM_ERROR, message.Op, "房间不存在")
			}
		case MatchUser:
			// 匹配用户
			b := CurrentServer.matchs.matchUser(c)
			if !b {
				c.SendError(MATCH_ERROR, message.Op, "已在匹配列表中")
			}
		case UpdateUserData:
			// 更新用户信息
			obj, bool := message.Data.(map[string]any)
			if bool {
				keys := make([]string, 0, len(obj))
				for k := range obj {
					keys = append(keys, k)
				}
				for _, v := range keys {
					c.userData[v] = obj[v]
				}
				c.SendToUserOp(&ClientMessage{
					Op: UpdateUserData,
				})
			} else {
				c.SendError(UPDATE_USER_ERROR, message.Op, "无效的操作数据")
			}
		case GetRoomOldMessage:
			// 获取房间的历史消息记录
			if c.room != nil {
				c.SendToUserOp(&ClientMessage{
					Op:   GetRoomOldMessage,
					Data: c.room.oldMsgs,
				})
			} else {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			}
		case UpdateRoomCustomData:
			// 更新房间自定义信息，房主操作
			if c.room == nil {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			} else {
				if c.room.master == c {
					c.room.updateCustomData(message.Data)
					c.SendToUserOp(&ClientMessage{
						Op: UpdateRoomCustomData,
					})
					c.room.onRoomChanged()
				} else {
					c.SendError(ROOM_PERMISSION_DENIED, message.Op, "需要房主操作")
				}
			}
		case UpdateRoomOption:
			// 更新房间的固定信息，人数、密码等
			if c.room == nil {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			} else {
				if c.room.master == c {
					m, b := message.Data.(map[string]any)
					if b {
						c.room.updateRoomData(RoomConfigOption{
							maxCounts: util.GetMapValueToInt(m, "maxCounts"),
							password:  util.GetMapValueToString(m, "password"),
						})
						c.SendToUserOp(&ClientMessage{
							Op: UpdateRoomOption,
						})
						c.room.onRoomChanged()
					} else {
						c.SendError(DATA_ERROR, message.Op, "数据结构错误")
					}
				} else {
					c.SendError(ROOM_PERMISSION_DENIED, message.Op, "需要房主操作")
				}
			}
		case KickOut:
			// 踢人流程
			if c.room == nil {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			} else {
				if c.room.master == c {
					m, b := message.Data.(map[string]any)
					if b {
						uid := util.GetMapValueToInt(m, "uid")
						if uid == c.room.master.uid {
							c.SendError(ROOM_PERMISSION_DENIED, message.Op, "无法踢出房主")
						} else {
							c.room.kickOut(uid)
						}
					} else {
						c.SendError(DATA_ERROR, message.Op, "数据结构错误")
					}
				} else {
					c.SendError(ROOM_PERMISSION_DENIED, message.Op, "需要房主操作")
				}
			}
		case GetFrameAt:
			// 获取范围帧数据 Bate
			if c.room == nil {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			} else {
				m, b := message.Data.(map[string]any)
				if b {
					start := util.GetMapValueToInt(m, "start")
					end := util.GetMapValueToInt(m, "end")
					if end == 0 {
						end = len(c.room.frameDatas)
					}
					c.SendToUserOp(&ClientMessage{
						Op:   GetFrameAt,
						Data: c.room.frameDatas[start:end],
					})
				} else {
					c.SendError(DATA_ERROR, message.Op, "数据结构错误")
				}
			}
		case SetRoomState:
			// 同步房间状态
			if c.room == nil {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			} else {
				m, b := message.Data.(map[string]any)
				if b {
					keys := make([]string, 0, len(m))
					for k := range m {
						keys = append(keys, k)
					}
					for _, v := range keys {
						c.room.roomState.Data[v] = m[v]
					}
					// 需要把更改数据下发给其他的所有人
					c.room.SendToAllUserOp(&ClientMessage{
						Op:   RoomStateUpdate,
						Data: m,
					}, c)
					// 通知更改成功
					c.SendToUserOp(&ClientMessage{
						Op: SetRoomState,
					})
				} else {
					c.SendError(DATA_ERROR, message.Op, "数据结构错误")
				}
			}
		case SetClientState:
			if c.room == nil {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			} else {
				m, b := message.Data.(map[string]any)
				if b {
					keys := make([]string, 0, len(m))
					for k := range m {
						keys = append(keys, k)
					}
					u, e := c.room.userState[c.uid]
					if !e || u == nil {
						u = &ClientState{
							Data: map[string]any{},
						}
					}
					for _, v := range keys {
						u.Data[v] = m[v]
					}
					c.room.userState[c.uid] = u
					// 需要把更改数据下发给其他的所有人
					c.room.SendToAllUserOp(&ClientMessage{
						Op: ClientStateUpdate,
						Data: map[string]any{
							"uid":  c.uid,
							"data": m,
						},
					}, c)
					fmt.Println("最后更改状态：", c.uid, u)
					// 通知更改成功
					c.SendToUserOp(&ClientMessage{
						Op: SetClientState,
					})
				} else {
					c.SendError(DATA_ERROR, message.Op, "数据结构错误")
				}
			}
		case ResetRoom:
			// 重置房间状态
			if c.room == nil {
				c.SendError(ROOM_NOT_EXSIT, message.Op, "房间不存在")
			} else {
				// 停止帧同步，同时清空帧缓存
				c.room.StopFrameSync()
				c.room.frameDatas = []any{}
				// 重置房间状态
				c.room.roomState = &ClientState{
					Data: map[string]any{},
				}
				// 重置用户状态
				c.room.userState = map[int]*ClientState{}
				c.SendToUserOp(&ClientMessage{
					Op: ResetRoom,
				})
			}
		default:
			c.SendError(OP_ERROR, message.Op, "无效的操作指令："+fmt.Sprint(message.Op))
		}
	} else {
		fmt.Println("处理命令失败", string(data), err.Error())
	}
}
