package models

import (
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/datastreams"
	"github.com/delta/dalal-street-server/utils"
)

func TestMain(m *testing.M) {
	conf := utils.GetConfiguration()
	utils.Init(conf)

	// TODO: remove this and insert MarketEvent data stream as a dependency to actual code, then insert stub while using it
	datastreams.Init(conf)
	datastreamsManager = datastreams.GetManager()

	// Initialize the models package, but don't call the Init method. It does extra things we don't
	// want to happen.
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "models",
	})
	// set the config global variable to the actual configuration
	config = conf

	os.Exit(m.Run())
}
