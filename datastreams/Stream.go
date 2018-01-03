package datastreams

import (
	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

type listener struct {
	update chan interface{}
	done   <-chan struct{}
}

var logger *logrus.Entry

func Init(config *utils.Config) {
	logger = utils.GetNewFileLogger("datastreams.log", 20, "debug", false).WithFields(logrus.Fields{
		"module": "datastreams",
	})
}

// StartStreams starts the data streams that run perpetually
func StartStreams() {
	go InitStockExchangeStream()
	go InitStockPricesStream()
}
