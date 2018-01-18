package matchingengine

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

type fakeMarketDepthStream struct {
	datastreams.MarketDepthStream
	sync.RWMutex
	TradeCount uint32
}

func (mds *fakeMarketDepthStream) AddTrade(price, qty uint32, createdAt string) {
	//mds.Lock()
	mds.TradeCount++
	//mds.Unlock()
}
func (mds *fakeMarketDepthStream) AddOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
}
func (mds *fakeMarketDepthStream) AddListener(done <-chan struct{}, update chan interface{}, sessionId string) {
}
func (mds *fakeMarketDepthStream) RemoveListener(sessionId string) {
}
func (mds *fakeMarketDepthStream) CloseOrder(isMarket bool, isAsk bool, price uint32, stockQuantity uint32) {
}
func (*fakeMarketDepthStream) run() {
}

func Benchmark_MatchingEngine(b *testing.B) {
	conf := utils.GetConfiguration()
	conf.LogLevel = "info"
	utils.Init(conf)

	fakeDepth := &fakeMarketDepthStream{}
	orderBook := NewOrderBook(1, fakeDepth)

	fillOrderFn = func(ask *models.Ask, bid *models.Bid, stockTradePrice uint32, stockTradeQty uint32) (askDone bool, bidDone bool, tr *models.Transaction) {
		askDone = (ask.StockQuantity - ask.StockQuantityFulfilled) == stockTradeQty
		bidDone = (bid.StockQuantity - bid.StockQuantityFulfilled) == stockTradeQty
		tr = &models.Transaction{
			StockQuantity: -int32(stockTradeQty),
		}
		return
	}

	go orderBook.StartStockMatching()

	b.ResetTimer()

	start := time.Now()

	for i := 0; i < b.N; i++ {
		isBuy := (i%2 == 0)
		var delta int
		if isBuy {
			delta = 1880
		} else {
			delta = 1884
		}

		// ASK 1893
		// ASK 1892
		// ASK 1891
		// ASK 1890
		// ASK 1889 crossable
		// ASK 1888 crossable
		// ASK 1887 crossable
		// ASK 1886 crossable
		// ASK 1885 crossable
		// ASK 1884 crossable

		// BID 1889 crossable
		// BID 1888 crossable
		// BID 1887 crossable
		// BID 1886 crossable
		// BID 1885 crossable
		// BID 1884 crossable
		// BID 1883
		// BID 1882
		// BID 1881
		// BID 1880

		price := delta + rand.Intn(10)
		qty := (rand.Intn(10) + 1) * 100

		if isBuy {
			orderBook.AddBidOrder(&models.Bid{
				Id:            uint32(i),
				UserId:        1,
				StockId:       1,
				OrderType:     models.Limit,
				Price:         uint32(price),
				StockQuantity: uint32(qty),
				CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			})
		} else {
			orderBook.AddAskOrder(&models.Ask{
				Id:            uint32(i),
				UserId:        2,
				StockId:       1,
				OrderType:     models.Limit,
				Price:         uint32(price),
				StockQuantity: uint32(qty),
				CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	diff := time.Now().Sub(start).Seconds()

	fakeDepth.RLock()
	b.Logf("Completed %d out of %d transactions in %.2fs", fakeDepth.TradeCount, b.N, diff)
	fakeDepth.RUnlock()
}
