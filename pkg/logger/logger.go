package logger

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 业务接口，与 logrus 解耦
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
}

// Field 结构化字段
type Field struct {
	Key   string
	Value any
}

// 将外部 Field 转为 logrus.Fields
func toLogrus(fields []Field) logrus.Fields {
	f := logrus.Fields{}
	for _, v := range fields {
		f[v.Key] = v.Value
	}
	return f
}

// 全局单例
var std Logger = &logrusAdapter{
	entry: logrus.NewEntry(logrus.StandardLogger()),
}

func SetLogger(l Logger) { std = l }
func L() Logger          { return std }

//-------------------- 适配器 --------------------//

type logrusAdapter struct {
	entry *logrus.Entry
}

func (a *logrusAdapter) Debug(msg string, fields ...Field) {
	a.entry.WithFields(toLogrus(fields)).Debug(msg)
}
func (a *logrusAdapter) Info(msg string, fields ...Field) {
	a.entry.WithFields(toLogrus(fields)).Info(msg)
}
func (a *logrusAdapter) Warn(msg string, fields ...Field) {
	a.entry.WithFields(toLogrus(fields)).Warn(msg)
}
func (a *logrusAdapter) Error(msg string, fields ...Field) {
	a.entry.WithFields(toLogrus(fields)).Error(msg)
}
func (a *logrusAdapter) Fatal(msg string, fields ...Field) {
	a.entry.WithFields(toLogrus(fields)).Fatal(msg)
}
func (a *logrusAdapter) WithFields(fields ...Field) Logger {
	return &logrusAdapter{entry: a.entry.WithFields(toLogrus(fields))}
}

// InitLogrus 初始化 logrus（可放 main 里）
func InitLogrus(level string) {
	lvl, _ := logrus.ParseLevel(level)
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	// 如需 JSON 格式：
	// logrus.SetFormatter(&logrus.JSONFormatter{})
}

func InitLogrusWithFile(level, path string) {
	lumber := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     7, // days
		Compress:   true,
	}
	logrus.SetOutput(lumber)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	lvl, _ := logrus.ParseLevel(level)
	logrus.SetLevel(lvl)
}
