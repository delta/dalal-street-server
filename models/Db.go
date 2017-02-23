// Package models handles everything between the database and the API.
// All business logic is written in this package, so the user of this
// package does not need to take care of race conditions in updating
// the data.
package models

import (
	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry
var DbOpen = utils.DbOpen

func init() {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "models",
	})

	LoadStocks()
	OpenMarket()
}
