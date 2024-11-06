/*
Copyright © 2024 Deepreo Siber Güvenlik A.S Resul ÇELİK <resul.celik@deepreo.com>
*/

package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Graylog2/go-gelf/gelf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type mode bool

const (
	DEV  mode = true
	PROD mode = false
)

func (m mode) Value() bool {
	return bool(m)
}

type Logger struct {
	logger *zap.Logger
}

var lg *Logger = nil

type LoggerConfig struct {
	Mode     mode   `json:"mode"`
	Graylog  bool   `json:"graylog"`
	GLogHost string `json:"glog_host"`
	GLogPort string `json:"glog_port"`
	FileW    io.Writer
}

func init() {
	if lg != nil {
		return
	}
	cfg := &LoggerConfig{
		Mode:    DEV,
		Graylog: false,
	}
	InitializeLogger(cfg)
}

func InitializeLogger(config *LoggerConfig) error {
	var loggers []zapcore.Core
	if config.Mode == DEV {
		level := zap.NewAtomicLevelAt(zap.DebugLevel)
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
		stdout := zapcore.AddSync(os.Stdout)
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		loggers = append(loggers, zapcore.NewCore(consoleEncoder, stdout, level))
	}
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	if config.Graylog {
		gw, err := gelf.NewWriter(fmt.Sprintf("%s:%s", config.GLogHost, config.GLogPort))
		if err != nil {
			return err
		}
		loggers = append(loggers, zapcore.NewCore(jsonEncoder, zapcore.AddSync(gw), level))
	} else if config.FileW != nil {
		loggers = append(loggers, zapcore.NewCore(jsonEncoder, zapcore.AddSync(config.FileW), level))
	} else if config.Mode == PROD {
		loggers = append(loggers, zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), level))
	}
	tree := zapcore.NewTee(loggers...)
	core := zapcore.NewSamplerWithOptions(
		tree,
		time.Second, // interval
		3,           // log first 3 entries
		0,           // thereafter log zero entires within the interval
	)
	logger := zap.New(core)
	lg = &Logger{logger: logger}
	return nil
}

func anyParser(v ...any) (string, []zapcore.Field) {
	message := ""
	zapFields := []zap.Field{}
	for _, val := range v {
		switch val := val.(type) {
		case string:
			message += (val + " ")
		case error:
			zapFields = append(zapFields, zap.Error(val))
		case zap.Field:
			zapFields = append(zapFields, val)
		}
	}
	return message, zapFields
}

func Info(v ...any) {
	msg, fields := anyParser(v...)
	lg.logger.Info(msg, fields...)
}

func Debug(v ...any) {
	msg, fields := anyParser(v...)
	lg.logger.Debug(msg, fields...)
}

func Warn(v ...any) {
	msg, fields := anyParser(v...)
	lg.logger.Warn(msg, fields...)
}

func Error(v ...any) {
	msg, fields := anyParser(v...)
	lg.logger.Error(msg, fields...)
}

func Fatal(v ...any) {
	msg, fields := anyParser(v...)
	lg.logger.Fatal(msg, fields...)
}

func Panic(v ...any) {
	msg, fields := anyParser(v...)
	lg.logger.Panic(msg, fields...)
}

func Sync() error {
	return lg.logger.Sync()
}
