package datastreams

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/delta/dalal-street-server/utils"
)

// StockExchangeStream interface defines the interface to interact with StockExchange stream
type StockExchangeStream interface {
	Run()
	SendStockExchangeUpdate(stockId uint32, dp *datastreams_pb.StockExchangeDataPoint)
	AddListener(done <-chan struct{}, updates chan interface{}, sessionId string)
	RemoveListener(sessionId string)
}

// stockExchangeStream implements the StockExchangeInterface
type stockExchangeStream struct {
	logger          *logrus.Entry
	broadcastStream BroadcastStream

	stockExchangeMutex    sync.RWMutex
	dirtyStocksInExchange map[uint32]*datastreams_pb.StockExchangeDataPoint
}

// newStockExchangeStream creates a new StockExchangeStream
func newStockExchangeStream() StockExchangeStream {
	return &stockExchangeStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.StockExchangeStream",
		}),
		broadcastStream:       NewBroadcastStream(),
		dirtyStocksInExchange: make(map[uint32]*datastreams_pb.StockExchangeDataPoint),
	}
}

// Run runs the StockExchangeStream. Call in a go func. Repeatedly broadcasts stockexchange
// data every 15s, if there's any update
func (ses *stockExchangeStream) Run() {
	var l = ses.logger.WithFields(logrus.Fields{
		"method": "Run",
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	for {
		ses.stockExchangeMutex.Lock()

		if len(ses.dirtyStocksInExchange) == 0 {
			ses.stockExchangeMutex.Unlock()
			l.Debugf("Nothing dirty yet. Sleeping for 15 seconds")
			time.Sleep(time.Second * 5)
			continue
		}

		l.Debugf("Found dirtyStocks")
		updateProto := &datastreams_pb.StockExchangeUpdate{
			StocksInExchange: ses.dirtyStocksInExchange,
		}
		ses.dirtyStocksInExchange = make(map[uint32]*datastreams_pb.StockExchangeDataPoint)
		ses.stockExchangeMutex.Unlock()

		ses.broadcastStream.BroadcastUpdate(updateProto)
		l.Debugf("Sent to %d listeners!. Sleeping for 15 seconds", ses.broadcastStream.GetListenersCount())

		time.Sleep(time.Second * 5)
	}
}

// SendStockExchangeUpdate records the update internally. It doesn't immediately send it.
// That's done by Run()
func (ses *stockExchangeStream) SendStockExchangeUpdate(stockId uint32, dp *datastreams_pb.StockExchangeDataPoint) {
	var l = ses.logger.WithFields(logrus.Fields{
		"method":        "SendStockExchangeUpdate",
		"param_stockId": stockId,
		"param_dp":      dp,
	})

	ses.stockExchangeMutex.Lock()
	ses.dirtyStocksInExchange[stockId] = dp
	ses.stockExchangeMutex.Unlock()

	l.Infof("Recorded update")
}

// AddListener adds a listener to the StockExchangeStream
func (ses *stockExchangeStream) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = ses.logger.WithFields(logrus.Fields{
		"method":          "AddListener",
		"param_sessionId": sessionId,
	})

	ses.broadcastStream.AddListener(sessionId, &listener{
		update: update,
		done:   done,
	})

	l.Infof("Added")
}

// RemoveListener removes a listener from the StockExchangeStream
func (ses *stockExchangeStream) RemoveListener(sessionId string) {
	var l = ses.logger.WithFields(logrus.Fields{
		"method":          "RemoveListener",
		"param_sessionId": sessionId,
	})

	ses.broadcastStream.RemoveListener(sessionId)

	l.Infof("Removed")
}
