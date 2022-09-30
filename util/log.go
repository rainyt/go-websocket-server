package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
)

const (
	// 控制输出日志信息的细节，不能控制输出的顺序和格式。
	// 输出的日志在每一项后会有一个冒号分隔：例如2009/01/23 01:23:23.123123 /a/b/c/d.go:23: message
	Ldate         = 1 << iota     // 日期：2009/01/23
	Ltime                         // 时间：01:23:23
	Lmicroseconds                 // 微秒级别的时间：01:23:23.123123（用于增强Ltime位）
	Llongfile                     // 文件全路径名+行号： /a/b/c/d.go:23
	Lshortfile                    // 文件名+行号：d.go:23（会覆盖掉Llongfile）
	LUTC                          // 使用UTC时间
	LstdFlags     = Ldate | Ltime // 标准logger的初始值
)

// 是否启动Log输出，如果设置为false，则不会有调试log
var EnableLog = true

func Log(data ...any) {
	if EnableLog {

		if _, file, line, ok := runtime.Caller(1); ok {
			log.Printf("%s:%d", file, line)
		}

		fmt.Println(data...)
	}
}

func InitLogger(model *string) {
	logFile, err := os.OpenFile("./console.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("打开日志文件错误:", err)
		return
	}

	if *model != "debug" {
		fmt.Println("日志输出到console.log文件")
		// log.SetOutput(logFile)

		// 组合一下即可，os.Stdout代表标准输出流
		//将日志输出到console.log和控制台
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multiWriter)
	} else {
		fmt.Println("日志输出到控制台")
	}

	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
}
