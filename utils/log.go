package utils

import (
	"os"

	"github.com/Sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is an instance of logrus.Logger
// Logger is to be used for all logging
var Logger *logrus.Logger

// initLogger initializes the logger with appropriate configuration options
func initLogger(config *Config) {
	var (
		fileName = config.LogFileName
		maxSize  = config.LogMaxSize
		logLevel = config.LogLevel
	)

	if config.LogFileName == "" {
		fileName = "./log.log"
	}

	if config.LogMaxSize == 0 {
		maxSize = 50
	}

	if config.LogLevel == "" {
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
	if fileName == "stdout" {
		Logger.Out = os.Stdout
		Logger.Formatter = &logrus.TextFormatter{}
	}

	Logger.Info("Logger started")
}

// GetNewFileLogger returns a logger that writes to a file of the given name
func GetNewFileLogger(fileName string, maxSize int, logLevel string, json bool) *logrus.Logger {
	if fileName == "" {
		fileName = "./log1.log"
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
