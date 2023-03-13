package runtime

import (
	"runtime"
	"websocket_server/util"
)

// 处理线程中报的错误，以避免引起主线程挂掉
func GoRecover() {
	if err := recover(); err != nil {
		util.Log(err)
		if _, file, line, ok := runtime.Caller(3); ok {
			util.Log("协程报错：%s:%d", file, line)
		}
	}
}
