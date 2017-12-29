package datastreams

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"
)

type Trade struct {
	TradeQuantity uint32
	TradePrice    uint32
	TradeTime     string
}

func (t *Trade) ToProto() *datastreams_pb.Trade {
	return &datastreams_pb.Trade{
		TradePrice:    t.TradePrice,
		TradeQuantity: t.TradeQuantity,
		TradeTime:     t.TradeTime,
	}
}

type MarketDepth struct {
	listenersLock sync.Mutex
	listeners     map[string]listener

	stockId uint32

	askDepthLock sync.Mutex
	askDepth     map[uint32]uint32
	askDepthDiff map[uint32]int32

	bidDepthLock sync.Mutex
	bidDepth     map[uint32]uint32
	bidDepthDiff map[uint32]int32

	latestTradesLock sync.Mutex
	latestTrades     []*Trade
	latestTradesDiff []*Trade
}

var marketDepthsMap = make(map[uint32]*MarketDepth)

func NewMarketDepth(stockId uint32) *MarketDepth {
	marketDepthsMap[stockId] = &MarketDepth{
		listeners:    make(map[string]listener),
		stockId:      stockId,
		askDepth:     make(map[uint32]uint32),
		askDepthDiff: make(map[uint32]int32),
		bidDepth:     make(map[uint32]uint32),
		bidDepthDiff: make(map[uint32]int32),
	}
	go marketDepthsMap[stockId].run()
	return marketDepthsMap[stockId]
}

func (md *MarketDepth) run() {
	var l = logger.WithFields(logrus.Fields{
		"method":        "MarketDepth.run",
		"param_stockId": md.stockId,
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	l.Infof("Started")

	for {
		md.listenersLock.Lock()
		l.Debugf("%d listeners as of now", len(md.listeners))
		if len(md.listeners) == 0 {
			l.Debugf("No listeners. Sleeping for 15 seconds")
			md.listenersLock.Unlock()
			time.Sleep(time.Minute / 4)
			continue
		}
		md.listenersLock.Unlock()

		var mdUpdate = &datastreams_pb.MarketDepthUpdate{
			StockId: md.stockId,
		}
		var shouldSend = false

		md.askDepthLock.Lock()
		if len(md.askDepthDiff) != 0 {
			mdUpdate.AskDepthDiff = md.askDepthDiff
			md.askDepthDiff = make(map[uint32]int32)
			shouldSend = true
		}
		md.askDepthLock.Unlock()

		md.bidDepthLock.Lock()
		if len(md.bidDepthDiff) != 0 {
			mdUpdate.BidDepthDiff = md.bidDepthDiff
			md.bidDepthDiff = make(map[uint32]int32)
			shouldSend = true
		}
		md.bidDepthLock.Unlock()

		md.latestTradesLock.Lock()
		if len(md.latestTradesDiff) != 0 {
			for _, t := range md.latestTradesDiff {
				mdUpdate.LatestTradesDiff = append(mdUpdate.LatestTradesDiff, t.ToProto())
			}
			md.latestTradesDiff = nil
			shouldSend = true
		}
		md.latestTradesLock.Unlock()

		if !shouldSend {
			l.Debugf("No update to send. Sleeping for 15 seconds")
			time.Sleep(time.Minute / 4)
			continue
		}

		sent := 0
		md.listenersLock.Lock()
		for sessionId, listener := range md.listeners {
			select {
			case <-listener.done:
				delete(md.listeners, sessionId)
				l.Debugf("Found dead listener. Removed")
			case listener.update <- mdUpdate:
				sent++
			}
		}
		md.listenersLock.Unlock()

		l.Debugf("Sent %+v to %d listeners! Sleeping for 15 seconds", mdUpdate, sent)
		time.Sleep(time.Minute / 4)
	}
}

func (md *MarketDepth) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method":          "MarketDepth.AddListener",
		"param_stockId":   md.stockId,
		"param_sessionId": sessionId,
	})

	md.listenersLock.Lock()
	defer md.listenersLock.Unlock()

	md.listeners[sessionId] = listener{
		update,
		done,
	}

	var mdUpdate = &datastreams_pb.MarketDepthUpdate{
		StockId: md.stockId,
	}

	md.askDepthLock.Lock()
	mdUpdate.AskDepth = md.askDepth
	md.askDepthLock.Unlock()

	md.bidDepthLock.Lock()
	mdUpdate.BidDepth = md.bidDepth
	md.bidDepthLock.Unlock()

	md.latestTradesLock.Lock()
	for _, t := range md.latestTrades {
		mdUpdate.LatestTrades = append(mdUpdate.LatestTrades, t.ToProto())
	}
	md.latestTradesLock.Unlock()

	l.Debugf("Sending %+v", mdUpdate)

	// Required to be done in a go-func, otherwise deadlock results. update chan isn't read until this function returns.
	go func() {
		select {
		case <-done:
			l.Debugf("Client exited before sending")
		case update <- mdUpdate:
			l.Debugf("Sent")
		}
	}()

	l.Debugf("Done")
}

func (md *MarketDepth) RemoveListener(sessionId string) {
	var l = logger.WithFields(logrus.Fields{
		"method":          "MarketDepth.RemoveListener",
		"param_stockId":   md.stockId,
		"param_sessionId": sessionId,
	})
	md.listenersLock.Lock()
	delete(md.listeners, sessionId)
	md.listenersLock.Unlock()
	l.Debugf("Removed")
}

func (md *MarketDepth) AddOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
	// Do not add Market orders to depth
	if isMarket {
		return
	}
	var l = logger.WithFields(logrus.Fields{
		"method":              "MarketDepth.AddOrder",
		"param_stockId":       md.stockId,
		"param_isMarket":      isMarket,
		"param_isAsk":         isAsk,
		"param_price":         price,
		"param_stockQuantity": stockQuantity,
	})

	l.Debugf("Adding")

	if isAsk {
		md.askDepthLock.Lock()
		md.askDepth[price] += stockQuantity
		md.askDepthDiff[price] += int32(stockQuantity)
		md.askDepthLock.Unlock()

		l.Debugf("Added")
		return
	}
	md.bidDepthLock.Lock()
	md.bidDepth[price] += stockQuantity
	md.bidDepthDiff[price] += int32(stockQuantity)
	md.bidDepthLock.Unlock()

	l.Debugf("Added")
}

func (md *MarketDepth) Trade(price, qty uint32, createdAt string) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "MarketDepth.Trade",
		"param_stockId": md.stockId,
		"param_price":   price,
		"param_qty":     qty,
	})

	t := &Trade{
		qty,
		price,
		createdAt,
	}

	md.latestTradesLock.Lock()
	if len(md.latestTrades) >= 10 {
		md.latestTrades = md.latestTrades[9-len(md.latestTrades):]
	}

	md.latestTrades = append(md.latestTrades, t)
	md.latestTradesDiff = append(md.latestTradesDiff, t)
	md.latestTradesLock.Unlock()

	l.Debugf("Added")
}

func (md *MarketDepth) CloseOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
	// Market orders have not even been added to depth
	if isMarket {
		return
	}
	if isAsk {
		md.askDepthLock.Lock()
		md.askDepth[price] -= stockQuantity
		md.askDepthDiff[price] -= int32(stockQuantity)
		if md.askDepth[price] == 0 {
			delete(md.askDepth, price)
		}
		if md.askDepthDiff[price] == 0 {
			delete(md.askDepthDiff, price)
		}
		md.askDepthLock.Unlock()
		return
	}
	md.bidDepthLock.Lock()
	md.bidDepth[price] -= stockQuantity
	md.bidDepthDiff[price] -= int32(stockQuantity)
	if md.bidDepth[price] == 0 {
		delete(md.bidDepth, price)
	}
	if md.bidDepthDiff[price] == 0 {
		delete(md.bidDepthDiff, price)
	}
	md.bidDepthLock.Unlock()
}

func RegMarketDepthListener(done <-chan struct{}, update chan interface{}, sessionId string, stockId uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":          "RegMarketDepthListener",
		"param_sessionId": sessionId,
		"param_stockId":   stockId,
	})

	l.Debugf("Got a listener")

	md, ok := marketDepthsMap[stockId]
	if !ok {
		l.Errorf("Invalid stock id")
		return fmt.Errorf("Invalid stockId")
	}

	md.AddListener(done, update, sessionId)
	go func() {
		<-done
		md.RemoveListener(sessionId)
		l.Debugf("Removed dead listener")
	}()

	return nil
}

func UnregMarketDepthListener(sessionId string, stockId uint32) {
	var l = logger.WithFields(logrus.Fields{
		"method":          "UnregMarketDepthListener",
		"param_sessionId": sessionId,
		"param_stockId":   stockId,
	})

	l.Debugf("Unregistering")

	md, ok := marketDepthsMap[stockId]
	if !ok {
		l.Errorf("Invalid stock id")
		return
	}

	md.RemoveListener(sessionId)
	l.Debugf("Done")
}
