package loging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	envLogLevel     = "LOG__LEVEL"
	defaultLogLevel = "INFO"
)

type Logger struct {
	*zap.Logger
}

func New(opts ...zap.Option) Logger {
	envLog := os.Getenv(envLogLevel)
	if envLog == "" {
		envLog = defaultLogLevel
	}

	var lv zapcore.Level
	err := lv.UnmarshalText([]byte(envLog))
	if err != nil {
		panic(err)
	}

	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(lv)
	loggerConfig.Development = false
	logger, _ := loggerConfig.Build(opts...)

	return Logger{logger}
}

func Err(err error) zap.Field {
	return zap.String("err", err.Error())
}
