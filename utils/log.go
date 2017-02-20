package utils

import (
	"github.com/Sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is an instance of logrus.Logger
// Logger is to be used for all logging
var Logger *logrus.Logger

// InitLogger initializes the logger with apropriate configuration options
func InitLogger() {
	var (
		fileName string = Configuration.LogFileName
		maxSize  int    = Configuration.LogMaxSize
		logLevel string = Configuration.LogLevel
	)

	if Configuration.LogFileName == "" {
		fileName = "./log.go"
	}

	if Configuration.LogMaxSize == 0 {
		maxSize = 50
	}

	if Configuration.LogLevel == "" {
		logLevel = "info"
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}

	Logger = &logrus.Logger{
		Formatter: &logrus.JSONFormatter{},
		Out: &lumberjack.Logger{
			Filename: fileName,
			MaxSize:  maxSize, // MB
		},
		Level: level,
	}

	Logger.Info("Logger started")
}

func GetNewFileLogger(fileName string, maxSize int, logLevel string, json bool) *logrus.Logger {
	if fileName == "" {
		fileName = "./log1.go"
	}

	if maxSize == 0 {
		maxSize = 50
	}

	if logLevel == "" {
		logLevel = "info"
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}

	logger := &logrus.Logger{
		Out: &lumberjack.Logger{
			Filename: fileName,
			MaxSize:  maxSize, // MB
		},
		Level: level,
	}

	if json {
		logger.Formatter = &logrus.JSONFormatter{}
	} else {
		logger.Formatter = &logrus.TextFormatter{}
	}

	return logger
}
