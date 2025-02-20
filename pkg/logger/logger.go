package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TODO traceID
// 为什么 fmt 在日志处理中可能不安全？
// 	1.	并发问题
// 	•	fmt 包中的函数（如 fmt.Printf、fmt.Fprintf 等）本身是线程安全的，但它们不会主动处理高并发日志写入的顺序问题。
// 	•	在高并发场景中，多个 goroutine 同时调用 fmt 输出日志时，可能导致日志内容错乱或丢失。
// 	2.	性能问题
// 	•	fmt 的性能相较于专门的日志库（如 zap）要低。
// 	•	zap 和类似日志库通常对性能进行了优化，比如减少不必要的内存分配，而 fmt 每次都会进行格式化和分配。
// 	3.	日志结构化缺失
// 	•	fmt 的输出通常是非结构化的，直接输出到控制台或者文件，不利于日志收集和分析。
// 	•	专业日志库（如 zap）支持结构化日志，方便在复杂系统中快速查询问题。

// LogInfo 日志实例
type LogInfo struct {
	log       *zap.Logger
	debugMode bool
	mu        sync.Mutex
}

var (
	// Log 日志实例
	Log *LogInfo
)

// var log = Initialize()

// Initialize 按日期分文件的日志初始化
func Initialize(serviceName, logDir string) *LogInfo {
	var (
		l = new(LogInfo)
	)
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create log directory: %v", err))
	}

	// 动态生成日志文件路径
	logFile := func() string {
		date := time.Now().Format("2006-01-02") // 每日日志文件名
		return filepath.Join(logDir, fmt.Sprintf("log_%s.log", date))
	}

	// 创建 INFO 和 ERROR 日志核心（输出到文件和控制台）
	infoErrorCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), // JSON 格式
		zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),                // 输出到控制台
			zapcore.AddSync(getFileWriter(logFile())), // 输出到动态文件
		),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= zap.InfoLevel && level <= zap.ErrorLevel
		}),
	)

	// 创建 DEBUG 日志核心（仅控制台，根据 debugMode 控制）
	debugCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), // 控制台友好格式
		zapcore.AddSync(os.Stdout),                                   // 输出到控制台
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			l.mu.Lock()
			defer l.mu.Unlock()
			return l.debugMode && level == zapcore.DebugLevel
		}),
	)

	// 创建 WARN 日志核心（仅控制台）
	warnCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), // 控制台友好格式
		zapcore.AddSync(os.Stdout),                                   // 输出到控制台
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.WarnLevel
		}),
	)

	// 合并日志核心
	core := zapcore.NewTee(infoErrorCore, debugCore, warnCore)

	// 创建日志实例
	l.log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return l
}

// getFileWriter 返回日志文件的 WriteSyncer
func getFileWriter(logFile string) zapcore.WriteSyncer {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open or create log file: %v", err))
	}
	return zapcore.AddSync(file)
}

// EnableDebugMode 启用 DEBUG 模式
func (l *LogInfo) EnableDebugMode() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugMode = true
}

// DisableDebugMode 禁用 DEBUG 模式
func (l *LogInfo) DisableDebugMode() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugMode = false
}

// SetDebugMode set the log debug mode
func (l *LogInfo) SetDebugMode(status bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugMode = status
}

// Info 输出 INFO 级别日志
func (l *LogInfo) Info(msg string, fields ...zap.Field) {
	l.log.Info(msg, fields...)
}

// Infof 输出 INFO 级别日志
func (l *LogInfo) Infof(msg string, info interface{}, fields ...zap.Field) {
	l.log.Info(fmt.Sprintf("%s %v", msg, info), fields...)
}

// Error 输出 ERROR 级别日志
func (l *LogInfo) Error(msg string, fields ...zap.Field) {
	l.log.Error(msg, fields...)
}

// Errorf 输出 ERROR 级别日志
func (l *LogInfo) Errorf(msg string, err error, fields ...zap.Field) {
	l.log.Error(fmt.Sprintf("%s %v", msg, err), fields...)
}

// Debug 输出 DEBUG 级别日志
func (l *LogInfo) Debug(msg string, fields ...zap.Field) {
	l.log.Debug(msg, fields...)
}

// Warn 输出 WARN 级别日志
func (l *LogInfo) Warn(msg string, fields ...zap.Field) {
	l.log.Warn(msg, fields...)
}

// Sync 刷新日志缓冲区（确保所有日志写入）
func (l *LogInfo) Sync() {
	if err := l.log.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
	}
}

// Fatal 输出错误日志并退出程序
func (l *LogInfo) Fatal(msg string, fields ...zap.Field) {
	l.log.Error(msg, fields...)
	// 尝试刷新日志缓冲区
	if err := l.log.Sync(); err != nil {
		os.Stderr.WriteString("Failed to sync logger: " + err.Error() + "\n")
	}
	l = new(LogInfo)
	// 退出程序
	os.Exit(1)
}

// os.Stderr 的核心特点
// 	1.	标准错误输出流
// 	•	os.Stderr 是操作系统提供的一个文件描述符，通常指向 终端或控制台。
// 	•	输出到 os.Stderr 的信息通常被认为是错误或警告信息。
// 	2.	与 os.Stdout 的区别
// 	•	os.Stdout：用于打印程序的正常输出，例如普通日志、计算结果等。
// 	•	os.Stderr：用于打印错误信息或调试信息，便于区分。
// 	3.	流分离的好处
// 	•	通过分别使用 os.Stdout 和 os.Stderr，可以让日志收集工具或操作系统轻松区分普通输出和错误输出。
