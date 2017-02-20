package datastreams

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	datastreams_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/datastreams"
)

var (
	stockPricesMutex sync.Mutex
	dirtyStocks      = make(map[uint32]uint32) // list of stocks whose updates we haven't sent yet

	stockPricesListenersMutex sync.Mutex
	stockPricesListeners      = make(map[string]listener)
)

func InitStockPricesStream() {
	var l = logger.WithFields(logrus.Fields{
		"method": "InitStockPricesStream",
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	for {
		stockPricesMutex.Lock()
		l.Debugf("%d listeners as of now", len(stockPricesListeners))
		if len(dirtyStocks) == 0 {
			stockPricesMutex.Unlock()
			l.Debugf("Nothing dirty yet. Sleeping for 15 seconds")
			time.Sleep(time.Minute / 4)
			continue
		}
		l.Debugf("Found dirty stock prices")
		updateProto := &datastreams_proto.StockPricesUpdate{
			Prices: dirtyStocks,
		}
		stockPricesMutex.Unlock()

		sent := 0
		stockPricesListenersMutex.Lock()
		l.Debugf("Will be sending %+v to %d listeners", updateProto, len(stockPricesListeners))
		for sessionId, listener := range stockPricesListeners {
			select {
			case <-listener.done:
				delete(stockPricesListeners, sessionId)
				l.Debugf("Found dead listener. Removed")
			case listener.update <- updateProto:
				sent++
			}
		}
		stockPricesListenersMutex.Unlock()

		l.Debugf("Sent to %d listeners! Sleeping for 15 seconds", sent)

		time.Sleep(time.Minute / 4)
	}
}

func SendStockPriceUpdate(stockId, price uint32) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "SendStockPriceUpdate",
		"param_stockId": stockId,
		"param_price":   price,
	})
	l.Debugf("Adding to the next stock prices update")
	stockPricesMutex.Lock()
	defer stockPricesMutex.Unlock()
	dirtyStocks[stockId] = price
}

func RegStockPricesListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method": "RegStockPricesListener",
	})

	l.Debugf("Got a listener")

	stockPricesListenersMutex.Lock()
	defer stockPricesListenersMutex.Unlock()

	if oldlistener, ok := stockPricesListeners[sessionId]; ok {
		// remove the old listener.
		close(oldlistener.update)
	}
	stockPricesListeners[sessionId] = listener{
		update,
		done,
	}
	go func() {
		<-done
		UnregStockPricesListener(sessionId)
		l.Debugf("Found dead listener. Removed")
	}()
}

func UnregStockPricesListener(sessionId string) {
	stockPricesListenersMutex.Lock()
	delete(stockPricesListeners, sessionId)
	stockPricesListenersMutex.Unlock()
}
