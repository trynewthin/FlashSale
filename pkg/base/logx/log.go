// logx 包包含相关应用代码。
package logx

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogConfig 定义日志初始化参数。
type LogConfig struct {
	Service string
	Level   string
	Format  string
}

// New 创建统一格式的 zap Logger。
func New(cfg LogConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if cfg.Level != "" {
		if err := level.UnmarshalText([]byte(strings.ToLower(cfg.Level))); err != nil {
			return nil, fmt.Errorf("invalid log level: %w", err)
		}
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encCfg)
	if strings.EqualFold(cfg.Format, "console") {
		encoder = zapcore.NewConsoleEncoder(encCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	logger := zap.New(core, zap.AddCaller())
	if cfg.Service != "" {
		logger = logger.With(zap.String("service", cfg.Service))
	}
	return logger, nil
}

// WithTrace 为日志实例附加 trace_id 字段。
func WithTrace(logger *zap.Logger, traceID string) *zap.Logger {
	if logger == nil {
		logger = zap.NewNop()
	}
	if traceID == "" {
		return logger
	}
	return logger.With(zap.String("trace_id", traceID))
}
