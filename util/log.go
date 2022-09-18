package util

import (
	"fmt"
)

// 是否启动Log输出，如果设置为false，则不会有调试log
var EnableLog = true

func Log(data ...any) {
	if EnableLog {
		fmt.Println(data...)
	}
}
