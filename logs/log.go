package logs

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

var openLog bool = true

func init() {
	if Log == nil {
		InitLogger("./log.log", zap.DebugLevel, true)
	}
}

// 日志开关
func OpenLog(isOpen bool) {
	openLog = isOpen
}

// tag: 日志的标记
// logpath： 日志文件地址
// level: 日志级别，如：zapcore.DebugLevel
func InitLogger(logpath string, level zapcore.Level, writeToConsole bool) {

	hook := lumberjack.Logger{
		Filename:   logpath, // 日志文件路径
		MaxSize:    128,     // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: 7,       // 日志文件最多保存多少个备份
		MaxAge:     7,       // 文件最多保存多少天
		Compress:   true,    // 是否压缩
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,                          // 小写编码器
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"), // 自定义时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,                         //
		// EncodeCaller:   zapcore.ShortCallerEncoder,     // 短路径编码器
		EncodeCaller: zapcore.FullCallerEncoder, // 全路径编码器
		EncodeName:   zapcore.FullNameEncoder,
	}

	// 设置日志级别
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(level)

	var writeSyncer zapcore.WriteSyncer

	if writeToConsole {
		writeSyncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)) // 打印到控制台
	} else {
		writeSyncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(&hook)) // 打印到文件
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig), // 控制台风格的编码器配置
		// zapcore.NewJSONEncoder(encoderConfig), // Json风格的编码器配置
		writeSyncer,
		atomicLevel, // 日志级别
	)

	// 开启开发模式，堆栈跟踪
	caller := zap.AddCaller()
	//跳过一层堆栈，避免行号指向封装器
	skip := zap.AddCallerSkip(1)

	// 开启文件及行号
	development := zap.Development()

	// 构造日志
	// Log = zap.New(core, caller, skip, development, zap.Fields(zap.String("tag", tag)))
	Log = zap.New(core, caller, skip, development)

	Log.Info("logs初始化成功")
	if writeToConsole {
		Log.Info("日志输出到控制台")
	} else {
		Log.Info("日志输出到文件:" + logpath)
	}
}

// 支持格式化
func InfoF(format string, a ...interface{}) {
	if openLog {
		Log.Info(fmt.Sprintf(format, a...))
	}
}

func DebugF(format string, a ...interface{}) {
	if openLog {
		Log.Debug(fmt.Sprintf(format, a...))
	}
}

func ErrorF(format string, a ...interface{}) {
	if openLog {
		Log.Error(fmt.Sprintf(format, a...))
	}
}

func FatalF(format string, a ...interface{}) {
	if openLog {
		Log.Fatal(fmt.Sprintf(format, a...))
	}
}

// 支持多参数
func InfoM(a ...interface{}) {
	if openLog {
		Log.Info(fmt.Sprint(a...))
	}
}

func DebugM(a ...interface{}) {
	if openLog {
		Log.Debug(fmt.Sprint(a...))
	}
}

func ErrorM(a ...interface{}) {
	if openLog {
		Log.Error(fmt.Sprint(a...))
	}
}

func FatalM(a ...interface{}) {
	if openLog {
		Log.Fatal(fmt.Sprint(a...))
	}
}

// 原始支持
func Info(msg string, fields ...zap.Field) {
	if openLog {
		Log.Info(msg, fields...)
	}
}

func Debug(msg string, fields ...zap.Field) {
	if openLog {
		Log.Debug(msg, fields...)
	}
}

func Error(msg string, fields ...zap.Field) {
	if openLog {
		Log.Error(msg, fields...)
	}
}

func Fatal(msg string, fields ...zap.Field) {
	if openLog {
		Log.Fatal(msg, fields...)
	}
}
