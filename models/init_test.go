package models

import (
	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

func init() {
	utils.InitConfiguration("config_test.json")
	utils.Logger = logrus.New()
	utils.Logger.Level = logrus.DebugLevel
}
