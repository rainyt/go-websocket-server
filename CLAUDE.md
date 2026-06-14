# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Golang 编写的 WebSocket 游戏服务器，提供房间管理、帧同步、状态同步、匹配等实时多人对战支持。对应的 Haxe 客户端 SDK：[hxonline](https://github.com/rainyt/hxonline)。

## 构建与运行

```bash
# 本地开发运行（debug 模式，日志输出到控制台）
go run ./main.go --port 8888 --ip 0.0.0.0 --wss 0 --model debug

# 生产模式（日志写入文件）
go run ./main.go --port 8888 --ip 0.0.0.0 --wss 0 --model product

# 启用 WSS（自动生成自签名 ECDSA 证书，或放置 tls.pem/tls.key 到根目录）
go run ./main.go --port 8888 --ip 0.0.0.0 --wss 1

# 交叉编译 Linux amd64 二进制
make build

# 部署到远程服务器
make upload
make run       # 在远程启动服务
make stop      # 停止远程服务
make log       # 查看远程日志
```

Go 版本要求：1.19+，模块名 `websocket_server`。

## 架构

### 分层结构

```
main.go                   → 入口，解析 CLI 参数，启动 net.Server
  net/                    → 核心业务层
    server.go             → Server（全局单例）→ App（按 appid 隔离）→ 用户/房间/匹配/全服消息
    client.go             → Client 结构体 + ClientAction 协议常量（47 个操作码）
    client_op_message.go  → OnMessage 消息路由，所有 OP 的 switch-case 处理
    room.go               → Room 生命周期：创建/加入/退出/帧同步/状态同步
    match.go              → 匹配系统：MatchOption 规则 + 匹配算法
    frame.go              → FrameData 结构体定义
    users.go              → UserDataSQL 用户身份管理（openId + userName 登录）
  websocketv2/            → WebSocket 连接层（gorilla/websocket 封装）
    websocket.go          → WebSocket 结构体，readMessage/writeMessage 双协程
    hub.go                → ServerHub，管理客户端注册/注销
  util/                   → 线程安全工具类
    map.go                → 带锁 Map + JSON 辅助函数（GetMapValueToInt/String/Any, SetJsonTo）
    array.go              → 带锁 Array
  logs/                   → 基于 zap + lumberjack 的日志（日志旋转、128MB 切割）
  runtime/                → GoRecover，goroutine panic 恢复
```

### 核心概念

**App 隔离**：通过 `appid` 将不同游戏/应用的用户、房间、匹配完全隔离。`Server.apps` 是 `map[string]*App`，每个 App 拥有独立的 `users`、`rooms`、`matchs`、`msglist`、`usersSQL`。

**用户登录**：客户端连接后必须先发送 `Login`（op=8），携带 `openid`、`username`、`appid`。同一 `openId` 重复登录会踢掉旧连接（发送 LOGIN_OUT_ERROR），然后将旧用户所在的房间无缝转移给新连接。

**帧同步**：房间锁定时运行，默认 30 FPS 间隔（`1./30.` 秒）。每帧收集所有用户缓存的 `FrameData`，打包为 `{t: cacheId, d: {uid: [frames...]}}` 下发给全房间。帧数据保存在 `Room.frameDatas` 数组中，支持按范围回查（`GetFrameAt`）。

**状态同步**：分两种 — 房间状态（`roomState`，所有人可读写）和用户状态（`userState`，每人独立）。更新时通知房间内其他成员，完整数据在 `GetRoomData` 时一起下发。

**匹配系统**：`MatchOption` 包含 `Key`（字符串相等匹配）、`Number`（目标人数）、`Range`（整数区间匹配）。匹配成功自动创建房间。

**扩展 API**：`Server.Register(api)` 通过反射注册扩展方法，方法名格式 `TypeName.MethodName`。客户端通过 `ExtendsCall`（op=42）携带 `{f: "TypeName.MethodName", d: ...}` 调用。`OnClosed` 方法会在客户端断开时自动触发。

### 通信协议

客户端与服务器通过 JSON 通信（也支持二进制首字节 OP + JSON body）。消息格式：`{op: int, data: any}`。所有操作码定义在 `net/client.go` 的 `ClientAction` 常量中（-1 到 46）。

### 关键依赖

- `github.com/gorilla/websocket` — WebSocket 底层
- `github.com/gin-gonic/gin` — HTTP 路由（仅用于升级 WebSocket，路由：`/`、`/hxonline`、`/hxonline/v2`）
- `github.com/json-iterator/go` — 高性能 JSON 序列化
- `go.uber.org/zap` + `gopkg.in/natefinch/lumberjack.v2` — 日志系统
