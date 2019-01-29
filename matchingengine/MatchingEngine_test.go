package matchingengine

import (
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/golang/mock/gomock"

	"github.com/delta/dalal-street-server/datastreams"
	"github.com/delta/dalal-street-server/mocks"
	"github.com/delta/dalal-street-server/models"
	"github.com/delta/dalal-street-server/utils"
)

func getMockMatchingEngine(t *testing.T) (
	*gomock.Controller,
	*mocks.MockOrderBook,
	*matchingEngine,
	uint32,
	uint64,
	uint64) {
	var stockID uint32 = 1
	var stockQuantity uint64 = 10
	var stockPrice uint64 = 20

	mockControl := gomock.NewController(t)

	mockDataStreamsManager := mocks.NewMockManager(mockControl)
	mockOrderBook := mocks.NewMockOrderBook(mockControl)

	mengine := &matchingEngine{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "matchingengine.MatchingEngine.test",
		}),
		orderBooks:         make(map[uint32]OrderBook),
		datastreamsManager: mockDataStreamsManager,
	}
	mengine.orderBooks[stockID] = mockOrderBook

	return mockControl, mockOrderBook, mengine, stockID, stockQuantity, stockPrice
}

func TestAddAskOrder(t *testing.T) {
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, mockOrderBook, mengine, stockID, stockQuantity, stockPrice := getMockMatchingEngine(t)
	defer mockControl.Finish()

	mengine.orderBooks[stockID] = mockOrderBook

	limitAsk := makeAsk(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockOrderBook.EXPECT().AddAskOrder(limitAsk)

	mengine.AddAskOrder(limitAsk)
}

func TestAddBidOrder(t *testing.T) {
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, mockOrderBook, mengine, stockID, stockQuantity, stockPrice := getMockMatchingEngine(t)
	defer mockControl.Finish()

	mengine.orderBooks[stockID] = mockOrderBook

	limitBid := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockOrderBook.EXPECT().AddBidOrder(limitBid)

	mengine.AddBidOrder(limitBid)
}

func TestCancelAskOrder(t *testing.T) {
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, mockOrderBook, mengine, stockID, stockQuantity, stockPrice := getMockMatchingEngine(t)
	defer mockControl.Finish()

	mengine.orderBooks[stockID] = mockOrderBook

	limitAsk := makeAsk(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockOrderBook.EXPECT().CancelAskOrder(limitAsk)

	mengine.CancelAskOrder(limitAsk)
}

func TestCancelBidOrder(t *testing.T) {
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, mockOrderBook, mengine, stockID, stockQuantity, stockPrice := getMockMatchingEngine(t)
	defer mockControl.Finish()

	mengine.orderBooks[stockID] = mockOrderBook

	limitBid := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockOrderBook.EXPECT().CancelBidOrder(limitBid)

	mengine.CancelBidOrder(limitBid)
}

func Test_LoadOldOrders(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_LoadOldOrders",
	})

	var stockID uint32 = 1
	// var stockQuantity uint64 = 10
	// var stockPrice uint64 = 20
	var userID1 uint32 = 3
	var userID2 uint32 = 4

	var makeAsk = func(userId uint32, stockId uint32, isClosed bool) *models.Ask {
		return &models.Ask{
			UserId:   userId,
			StockId:  stockId,
			IsClosed: isClosed,
		}
	}

	var makeBid = func(userId uint32, stockId uint32, isClosed bool) *models.Bid {
		return &models.Bid{
			UserId:   userId,
			StockId:  stockId,
			IsClosed: isClosed,
		}
	}

	stock := &models.Stock{
		Id: 1,
	}

	user1 := &models.User{Id: userID1, Cash: 2000000, Email: "test1@test.com"}
	user2 := &models.User{Id: userID2, Cash: 2000000, Email: "test2@test.com"}

	ask := makeAsk(3, 1, false)
	bid := makeBid(4, 1, false)

	db := utils.GetDB()

	defer func() {
		db.Delete(ask)
		db.Delete(bid)
		db.Delete(user1)
		db.Delete(user2)
		db.Delete(stock)
	}()

	if err := db.Create(user1).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(user2).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(stock).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(ask).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(bid).Error; err != nil {
		t.Fatal(err)
	}

	testAskQueue := NewAskPQueue(1)
	testBidQueue := NewBidPQueue(0)
	testAskStoplossQueue := NewAskPQueue(1)
	testBidStoplossQueue := NewBidPQueue(0)
	testDepth := datastreams.GetManager().GetMarketDepthStream(stockID)

	ob := &orderBook{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module":        "matchingengine.OrderBook.test",
			"param_stockId": stockID,
		}),
		stockId:     stockID,
		askChan:     make(chan *models.Ask),
		bidChan:     make(chan *models.Bid),
		asks:        testAskQueue,
		bids:        testBidQueue,
		askStoploss: testAskStoplossQueue,
		bidStoploss: testBidStoplossQueue,
		depth:       testDepth,
	}

	go ob.LoadOldAsk(ask)
	go ob.LoadOldBid(bid)
	time.Sleep(time.Second * 2)

	if testAskQueue.Head() != ask || testBidQueue.Head() != bid {
		l.Errorf("Error in testLoadOrders")
	}

}
