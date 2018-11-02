package datastreams

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/delta/dalal-street-server/utils"
)

// StockPricesStream interface defines the interface to interact with StockPrices stream
type StockPricesStream interface {
	Run()
	SendStockPriceUpdate(stockId uint32, price uint32)
	AddListener(done <-chan struct{}, updates chan interface{}, sessionId string)
	RemoveListener(sessionId string)
}

// stockPricesStream implements StockPricesStream interface
type stockPricesStream struct {
	logger          *logrus.Entry
	broadcastStream BroadcastStream

	stockPricesMutex sync.RWMutex
	dirtyStocks      map[uint32]uint32 // list of stocks whose updates we haven't sent yet
}

// newStockPricesStream creates a new StockPricesStream
func newStockPricesStream() StockPricesStream {
	return &stockPricesStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.StockPricesStream",
		}),
		broadcastStream: NewBroadcastStream(),
		dirtyStocks:     make(map[uint32]uint32),
	}
}

// Run runs the StockPricesStream. Call in a gofunc. Repeatedly broadcasts stock prices
// every 15 seconds, if there's any update
func (sps *stockPricesStream) Run() {
	var l = sps.logger.WithFields(logrus.Fields{
		"method": "Run",
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	for {
		sps.stockPricesMutex.Lock()

		if len(sps.dirtyStocks) == 0 {
			sps.stockPricesMutex.Unlock()
			l.Debugf("Nothing dirty yet. Sleeping for 15 seconds")
			time.Sleep(time.Second * 5)
			continue
		}

		l.Debugf("Found dirty stock prices")
		updateProto := &datastreams_pb.StockPricesUpdate{
			Prices: sps.dirtyStocks,
		}
		sps.dirtyStocks = make(map[uint32]uint32)
		sps.stockPricesMutex.Unlock()

		sps.broadcastStream.BroadcastUpdate(updateProto)

		l.Debugf("Sent to %d listeners! Sleeping for 15 seconds", sps.broadcastStream.GetListenersCount())

		time.Sleep(time.Second * 5)
	}
}

// SendStockPriceUpdate updates price of a given stock. It doesn't send it immediately. That's done by Run.
func (sps *stockPricesStream) SendStockPriceUpdate(stockId, price uint32) {
	var l = sps.logger.WithFields(logrus.Fields{
		"method":        "SendStockPriceUpdate",
		"param_stockId": stockId,
		"param_price":   price,
	})

	l.Debugf("Adding to the next stock prices update")
	sps.stockPricesMutex.Lock()
	sps.dirtyStocks[stockId] = price
	sps.stockPricesMutex.Unlock()
}

// AddListener adds a listener to the StockPricesStream
func (sps *stockPricesStream) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = sps.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_sessionId": sessionId,
	})

	sps.broadcastStream.AddListener(sessionId, &listener{
		update: update,
		done:   done,
	})

	l.Infof("Added")
}

// RemoveListener removes a listener from the StockPricesStream
func (sps *stockPricesStream) RemoveListener(sessionId string) {
	var l = sps.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_sessionId": sessionId,
	})

	sps.RemoveListener(sessionId)

	l.Infof("Removed")
}
