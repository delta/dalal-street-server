// Package models handles everything between the database and the API.
// All business logic is written in this package, so the user of this
// package does not need to take care of race conditions in updating
// the data.
package models

import (
	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/datastreams"
	"github.com/delta/dalal-street-server/utils"
)

var logger *logrus.Entry
var getDB = utils.GetDB
var config *utils.Config
var datastreamsManager datastreams.Manager

// Init configures the models package
func Init(conf *utils.Config, dsm datastreams.Manager) {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "models",
	})

	config = conf
	datastreamsManager = dsm

	LoadStocks()
}
