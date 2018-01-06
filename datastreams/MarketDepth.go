package datastreams

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// MarketDepthStream defines the interface for accessing a single stock's market depth
type MarketDepthStream interface {
	run()
	AddListener(done <-chan struct{}, updates chan interface{}, sessionId string)
	RemoveListener(sessionId string)

	AddOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32)
	AddTrade(price, qty uint32, createdAt string)
	CloseOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) // To be called after trades as well
}

// trade represents a single trade for a given stock
type trade struct {
	tradeQuantity uint32
	tradePrice    uint32
	tradeTime     string
}

// ToProto converts trade to its proto representation
func (t *trade) ToProto() *datastreams_pb.Trade {
	return &datastreams_pb.Trade{
		TradePrice:    t.tradePrice,
		TradeQuantity: t.tradeQuantity,
		TradeTime:     t.tradeTime,
	}
}

// marketDepthStream implements the MarketDepthStream interface
type marketDepthStream struct {
	logger *logrus.Entry

	stockId uint32

	askDepthLock sync.Mutex
	askDepth     map[uint32]uint32
	askDepthDiff map[uint32]int32

	bidDepthLock sync.Mutex
	bidDepth     map[uint32]uint32
	bidDepthDiff map[uint32]int32

	latestTradesLock sync.Mutex
	latestTrades     []*trade
	latestTradesDiff []*trade

	broadcastStream BroadcastStream
}

// newMarketDepthStream returns a new MarketDepthStream instance for a given stockId
func newMarketDepthStream(stockId uint32) MarketDepthStream {
	mds := &marketDepthStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module":        "datastreams.MarketDepthStream",
			"param_stockId": stockId,
		}),
		stockId:      stockId,
		askDepth:     make(map[uint32]uint32),
		askDepthDiff: make(map[uint32]int32),
		bidDepth:     make(map[uint32]uint32),
		bidDepthDiff: make(map[uint32]int32),

		broadcastStream: NewBroadcastStream(),
	}

	go mds.run()
	return mds
}

// run runs the marketDepthStream by updating the listeners ever 15 seconds
func (mds *marketDepthStream) run() {
	var l = mds.logger.WithFields(logrus.Fields{
		"method": "run",
	})

	defer func() {
		if r := recover(); r != nil {
			l.Errorf("Error! Stack trace: %s", string(debug.Stack()))
		}
	}()

	l.Infof("Started")

	for {
		if mds.broadcastStream.GetListenersCount() == 0 {
			l.Debugf("No listeners. Sleeping for 15 seconds")
			time.Sleep(time.Minute / 4)
			continue
		}

		var mdUpdate = &datastreams_pb.MarketDepthUpdate{
			StockId: mds.stockId,
		}
		var shouldSend = false

		mds.askDepthLock.Lock()
		if len(mds.askDepthDiff) != 0 {
			mdUpdate.AskDepthDiff = mds.askDepthDiff
			mds.askDepthDiff = make(map[uint32]int32)
			shouldSend = true
		}
		mds.askDepthLock.Unlock()

		mds.bidDepthLock.Lock()
		if len(mds.bidDepthDiff) != 0 {
			mdUpdate.BidDepthDiff = mds.bidDepthDiff
			mds.bidDepthDiff = make(map[uint32]int32)
			shouldSend = true
		}
		mds.bidDepthLock.Unlock()

		mds.latestTradesLock.Lock()
		if len(mds.latestTradesDiff) != 0 {
			for _, t := range mds.latestTradesDiff {
				mdUpdate.LatestTradesDiff = append(mdUpdate.LatestTradesDiff, t.ToProto())
			}
			mds.latestTradesDiff = nil
			shouldSend = true
		}
		mds.latestTradesLock.Unlock()

		if !shouldSend {
			l.Debugf("No update to send. Sleeping for 15 seconds")
			time.Sleep(time.Minute / 4)
			continue
		}

		mds.broadcastStream.BroadcastUpdate(mdUpdate)
		l.Debugf("Sent %+v to %d listeners! Sleeping for 15 seconds", mdUpdate, mds.broadcastStream.GetListenersCount())
		time.Sleep(time.Minute / 4)
	}
}

// AddListener adds a listener to marketDepthStream
func (mds *marketDepthStream) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
	var l = mds.logger.WithFields(logrus.Fields{
		"method":          "addListener",
		"param_sessionId": sessionId,
	})

	var mdUpdate = &datastreams_pb.MarketDepthUpdate{
		StockId: mds.stockId,
	}

	mds.askDepthLock.Lock()
	mdUpdate.AskDepth = mds.askDepth
	mds.askDepthLock.Unlock()

	mds.bidDepthLock.Lock()
	mdUpdate.BidDepth = mds.bidDepth
	mds.bidDepthLock.Unlock()

	mds.latestTradesLock.Lock()
	for _, t := range mds.latestTrades {
		mdUpdate.LatestTrades = append(mdUpdate.LatestTrades, t.ToProto())
	}
	mds.latestTradesLock.Unlock()

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

// RemoveListener removes a listener from a marketDepthStream
func (mds *marketDepthStream) RemoveListener(sessionId string) {
	var l = mds.logger.WithFields(logrus.Fields{
		"method":          "removeListener",
		"param_sessionId": sessionId,
	})
	mds.broadcastStream.RemoveListener(sessionId)
	l.Infof("Removed")
}

// AddOrder adds an order to a marketdepth stream
func (mds *marketDepthStream) AddOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
	// Do not add Market orders to depth
	if isMarket {
		return
	}

	var l = mds.logger.WithFields(logrus.Fields{
		"method":              "MarketDepth.AddOrder",
		"param_isMarket":      isMarket,
		"param_isAsk":         isAsk,
		"param_price":         price,
		"param_stockQuantity": stockQuantity,
	})

	l.Debugf("Adding")

	if isAsk {
		mds.askDepthLock.Lock()
		mds.askDepth[price] += stockQuantity
		mds.askDepthDiff[price] += int32(stockQuantity)
		mds.askDepthLock.Unlock()

		l.Debugf("Added")
		return
	}
	mds.bidDepthLock.Lock()
	mds.bidDepth[price] += stockQuantity
	mds.bidDepthDiff[price] += int32(stockQuantity)
	mds.bidDepthLock.Unlock()

	l.Debugf("Added")
}

// AddTrade adds a trade to the marketdepthstream
func (mds *marketDepthStream) AddTrade(price, qty uint32, createdAt string) {
	var l = mds.logger.WithFields(logrus.Fields{
		"method":      "MarketDepth.Trade",
		"param_price": price,
		"param_qty":   qty,
	})

	t := &trade{
		qty,
		price,
		createdAt,
	}

	mds.latestTradesLock.Lock()
	if len(mds.latestTrades) >= 20 {
		// FIXME: Check if this is correct and won't throw any error
		mds.latestTrades = mds.latestTrades[len(mds.latestTrades)-20:]
	}

	mds.latestTrades = append(mds.latestTrades, t)
	mds.latestTradesDiff = append(mds.latestTradesDiff, t)
	mds.latestTradesLock.Unlock()

	l.Debugf("Added")
}

// CloseOrder will close an order from the order book. It should be called every time an order closes
// either due to cancellation or due to fulfillment
func (mds *marketDepthStream) CloseOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
	// Market orders have not even been added to depth
	if isMarket {
		return
	}
	if isAsk {
		mds.askDepthLock.Lock()
		mds.askDepth[price] -= stockQuantity
		mds.askDepthDiff[price] -= int32(stockQuantity)
		if mds.askDepth[price] == 0 {
			delete(mds.askDepth, price)
		}
		if mds.askDepthDiff[price] == 0 {
			delete(mds.askDepthDiff, price)
		}
		mds.askDepthLock.Unlock()
		return
	}
	mds.bidDepthLock.Lock()
	mds.bidDepth[price] -= stockQuantity
	mds.bidDepthDiff[price] -= int32(stockQuantity)
	if mds.bidDepth[price] == 0 {
		delete(mds.bidDepth, price)
	}
	if mds.bidDepthDiff[price] == 0 {
		delete(mds.bidDepthDiff, price)
	}
	mds.bidDepthLock.Unlock()
}
