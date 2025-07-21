package logger

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/syseng-agent/pkg/types"
)

var log *logrus.Logger

func Init(config *types.Config) {
	log = logrus.New()
	
	level, err := logrus.ParseLevel(config.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
	
	switch config.Logging.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{})
	default:
		log.SetFormatter(&logrus.JSONFormatter{})
	}
	
	log.SetOutput(os.Stdout)
}

func GetLogger() *logrus.Logger {
	if log == nil {
		log = logrus.New()
	}
	return log
}

func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}