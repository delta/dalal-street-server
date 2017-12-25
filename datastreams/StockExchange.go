package datastreams

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"
)

var (
	dirtyStocksInExchange = make(map[uint32]*datastreams_pb.StockExchangeDataPoint) // list of stocks whose updates we haven't sent yet
	stockExchangeMutex    sync.Mutex

	stockExchangeListenersMutex sync.Mutex
	stockExchangeListeners      = make(map[string]listener)
)

func InitStockExchangeStream() {
	var l = logger.WithFields(logrus.Fields{
		"method": "InitStockExchangeStream",
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	for {
		stockExchangeMutex.Lock()

		stockExchangeListenersMutex.Lock()
		l.Debugf("%d listeners as of now", len(stockExchangeListeners))
		stockExchangeListenersMutex.Unlock()

		if len(dirtyStocksInExchange) == 0 {
			stockExchangeMutex.Unlock()
			l.Debugf("Nothing dirty yet. Sleeping for 15 seconds")
			time.Sleep(time.Minute / 4)
			continue
		}
		l.Debugf("Found dirtyStocks")
		updateProto := &datastreams_pb.StockExchangeUpdate{
			StocksInExchange: dirtyStocksInExchange,
		}
		dirtyStocksInExchange = make(map[uint32]*datastreams_pb.StockExchangeDataPoint)
		stockExchangeMutex.Unlock()

		sent := 0
		stockExchangeListenersMutex.Lock()
		l.Debugf("Will be sending %+v to %d listeners", updateProto, len(stockExchangeListeners))

		for sessionId, listener := range stockExchangeListeners {
			select {
			case <-listener.done:
				delete(stockExchangeListeners, sessionId)
				l.Debugf("Found dead listener. Removed")
			case listener.update <- updateProto:
				sent++
			}
		}

		stockExchangeListenersMutex.Unlock()

		l.Debugf("Sent to %d listeners!. Sleeping for 15 seconds", sent)

		time.Sleep(time.Minute / 4)
	}
}

func SendStockExchangeUpdate(stockId, price, stocksInExchange, stocksInMarket uint32) {
	var l = logger.WithFields(logrus.Fields{
		"method":                 "SendStockExchangeUpdate",
		"param_stockId":          stockId,
		"param_stocksInExchange": stocksInExchange,
		"param_stocksInMarket":   stocksInMarket,
	})

	l.Debugf("Adding to the next stock exchange update")

	stockExchangeMutex.Lock()
	defer stockExchangeMutex.Unlock()
	dirtyStocksInExchange[stockId] = &datastreams_pb.StockExchangeDataPoint{
		Price:            price,
		StocksInExchange: stocksInExchange,
		StocksInMarket:   stocksInMarket,
	}
}

func RegStockExchangeListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method": "RegStockExchangeListener",
	})
	l.Debugf("Got a listener")

	stockExchangeListenersMutex.Lock()
	defer stockExchangeListenersMutex.Unlock()

	if oldlistener, ok := stockExchangeListeners[sessionId]; ok {
		// remove the old listener.
		close(oldlistener.update)
	}
	stockExchangeListeners[sessionId] = listener{
		update,
		done,
	}

	go func() {
		<-done
		UnregStockExchangeListener(sessionId)
		l.Debugf("Found dead listener. Removed")
	}()
}

func UnregStockExchangeListener(sessionId string) {
	stockExchangeListenersMutex.Lock()
	delete(stockExchangeListeners, sessionId)
	stockExchangeListenersMutex.Unlock()
}
