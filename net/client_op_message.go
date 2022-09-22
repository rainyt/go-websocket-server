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
		if c.uid == 0 {
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
					c.SendError(LOGIN_ERROR, message.Op, "已登陆")
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
			room := CurrentServer.CreateRoom(c)
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
		case GetRoomMessage:
			// 获取房间信息
			if c.room != nil {
				c.SendToUserOp(&ClientMessage{
					Op:   GetRoomMessage,
					Data: c.room.GetRoomData(),
				})
			} else {
				c.SendError(GET_ROOM_ERROR, message.Op, "不存在房间信息")
			}
		case JoinRoom:
			if c.room != nil {
				c.SendError(JOIN_ROOM_ERROR, message.Op, "已存在房间，无法加入")
			} else {
				id := message.Data.(map[string]any)["id"]
				util.Log("加入房间", id)
				room, b := CurrentServer.JoinRoom(c, int(id.(float64)))
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
				c.room.SendToAllUserOp(&message, c)
				c.SendToUserOp(&ClientMessage{
					Op: RoomMessage,
				})
			} else {
				c.SendError(SEND_ROOM_ERROR, message.Op, "房间不存在")
			}
		default:
			c.SendError(OP_ERROR, message.Op, "无效的操作指令："+fmt.Sprint(message.Op))
		}
	} else {
		fmt.Println("处理命令失败", string(data), err.Error())
	}
}
