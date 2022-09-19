package net

import (
	"encoding/json"
	"fmt"
	"websocket_server/util"
)

// 消息处理
func (c *Client) onMessage(data []byte) {
	if !c.frameIsBinary {
		// 解析API操作
		message := ClientMessage{}
		err := json.Unmarshal(data, &message)
		if err == nil {
			fmt.Println("处理命令", message)
			if c.uid == 0 {
				switch message.Op {
				case Login:
					if c.uid == 0 {
						loginData := message.Data.(map[string]any)
						userName := loginData["username"]
						openId := loginData["openid"]
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
						c.SendError(LOGIN_ERROR, "已登陆")
					}
				default:
					c.SendError(OP_ERROR, "无效的操作指令："+fmt.Sprint(message.Op))
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
					c.SendError(CREATE_ROOM_ERROR, "房间已存在，无法创建")
				}
			case GetRoomMessage:
				// 获取房间信息
				if c.room != nil {
					c.SendToUserOp(&ClientMessage{
						Op:   GetRoomMessage,
						Data: c.room.GetRoomData(),
					})
				} else {
					c.SendError(GET_ROOM_ERROR, "不存在房间信息")
				}
			case StartFrameSync:
				// 开始帧同步
				if c.room != nil {
					c.room.StartFrameSync()
					c.SendToUserOp(&ClientMessage{
						Op: StartFrameSync,
					})
				} else {
					c.SendError(START_FRAME_SYNC_ERROR, "房间不存在，无法启动帧同步")
				}
			case StopFrameSync:
				// 开始停止帧同步
				if c.room != nil {
					c.room.StopFrameSync()
					c.SendToUserOp(&ClientMessage{
						Op: StopFrameSync,
					})
				} else {
					c.SendError(STOP_FRAME_SYNC_ERROR, "房间不存在，无法停止帧同步")
				}
			case UploadFrame:
				if c.room != nil && c.room.frameSync {
					// 缓存到用户数据中
					fdata, err := message.Data.(FrameData)
					if err {
						// 验证是否操作数据是否已无效
						if !isInvalidData(&fdata) {
							c.frames.Push(fdata)
							c.SendToUserOp(&ClientMessage{
								Op: UploadFrame,
							})
						} else {
							c.SendError(UPLOAD_FRAME_ERROR, "上传帧数据时间戳错误")
						}
					}
				} else {
					c.SendError(UPLOAD_FRAME_ERROR, "上传帧同步数据错误")
				}
			default:
				c.SendError(OP_ERROR, "无效的操作指令："+fmt.Sprint(message.Op))
			}
		} else {
			fmt.Println("处理命令失败", string(data))
		}
	}
}
