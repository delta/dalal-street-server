package matchingengine

import (
	"math/rand"
	"testing"
	"time"

	"github.com/thakkarparth007/dalal-street-server/models"
)

func Benchmark_MatchingEngine(b *testing.B) {
	db, err := DbOpen()
	stock := &Stock{
		Id:           1,
		CurrentPrice: 2000,
	}
	db.Save(stock)
	defer db.Delete(stock)
	models.LoadStocks()
	fakeDepth := NewFakeMarketDepthStream(1)
	fakeEngine := NewMatchingEngine(fakeDepth)

	orderBook := NewOrderBook(1, fakeDepth)

	r := rand.New(time.Now().UnixNano())

	for i := 0; i < b.N; i++ {
		isBuy := i % 2
		var delta uint32
		if isBuy {
			delta = 1880
		} else {
			delta = 1884
		}

		price := r.Int(10)
	}

}
