package net

import "time"

type FrameData struct {
	Time int64 // 时间戳
	Data any   // 帧数据
}

// 检验是否为无效数据，当操作数据超过1秒后的执行，则统一无效处理
func isInvalidData(data *FrameData) bool {
	current := time.Now().Unix()
	return current-data.Time > 1
}
