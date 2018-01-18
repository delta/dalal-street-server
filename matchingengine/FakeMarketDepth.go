package matchingengine

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

type fakeMarketDepthStream struct {
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

	TradeCount uint32

	broadcastStream datastreams.BroadcastStream
}

type trade struct {
	tradeQuantity uint32
	tradePrice    uint32
	tradeTime     string
}

func NewFakeMarketDepthStream(stockId uint32) datastreams.MarketDepthStream {
	mds := &fakeMarketDepthStream{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module":        "datastreams.MarketDepthStream",
			"param_stockId": stockId,
		}),
		stockId:      stockId,
		askDepth:     make(map[uint32]uint32),
		askDepthDiff: make(map[uint32]int32),
		bidDepth:     make(map[uint32]uint32),
		bidDepthDiff: make(map[uint32]int32),
		TradeCount:   0,
	}

	return mds
}

func (mds *fakeMarketDepthStream) AddOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
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

func (mds *fakeMarketDepthStream) AddTrade(price, qty uint32, createdAt string) {
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

	mds.TradeCount++

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

func (mds *fakeMarketDepthStream) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
}

func (mds *fakeMarketDepthStream) RemoveListener(sessionId string) {
}

func (mds *fakeMarketDepthStream) CloseOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
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

func (*fakeMarketDepthStream) run() {

}
