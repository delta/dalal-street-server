package matchingengine

import (
	"fmt"
	"testing"
	"time"

	"github.com/delta/dalal-street-server/datastreams"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/mocks"
	"github.com/delta/dalal-street-server/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/golang/mock/gomock"
)

func helperForOrderBookTests(t *testing.T) (
	*gomock.Controller,
	*orderBook,
	*mocks.MockAskPQueue,
	*mocks.MockBidPQueue,
	*mocks.MockAskPQueue,
	*mocks.MockBidPQueue,
	*mocks.MockMarketDepthStream,
	uint32,
	uint64,
	uint64) {
	// This initializes the logger. This is required because LoadOldAsk tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	// config := utils.GetConfiguration()
	// utils.Init(config)

	mockControl := gomock.NewController(t)
	// defer mockControl.Finish()

	var stockID uint32 = 1
	var stockQuantity uint64 = 10
	var stockPrice uint64 = 20

	mockAskQueue := mocks.NewMockAskPQueue(mockControl)
	mockBidQueue := mocks.NewMockBidPQueue(mockControl)
	mockAskStoplossQueue := mocks.NewMockAskPQueue(mockControl)
	mockBidStoplossQueue := mocks.NewMockBidPQueue(mockControl)
	mockDepth := mocks.NewMockMarketDepthStream(mockControl)

	ob := &orderBook{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module":        "matchingengine.OrderBook.test",
			"param_stockId": stockID,
		}),
		stockId:     stockID,
		askChan:     make(chan *models.Ask),
		bidChan:     make(chan *models.Bid),
		asks:        mockAskQueue,
		bids:        mockBidQueue,
		askStoploss: mockAskStoplossQueue,
		bidStoploss: mockBidStoplossQueue,
		depth:       mockDepth,
	}

	return mockControl, ob, mockAskQueue, mockBidQueue, mockAskStoplossQueue, mockBidStoplossQueue, mockDepth, stockID, stockQuantity, stockPrice
}

func helperForOrderBookTestsWithoutMockPQ(t *testing.T) (
	*gomock.Controller,
	*orderBook,
	*AskPQueue,
	*BidPQueue,
	*AskPQueue,
	*BidPQueue,
	*mocks.MockMarketDepthStream,
	uint32,
	uint64,
	uint64) {
	// This initializes the logger. This is required because LoadOldAsk tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	// config := utils.GetConfiguration()
	// utils.Init(config)

	mockControl := gomock.NewController(t)
	// defer mockControl.Finish()

	var stockID uint32 = 1
	var stockQuantity uint64 = 10
	var stockPrice uint64 = 20

	testAskQueue := NewAskPQueue(1)
	testBidQueue := NewBidPQueue(0)
	testAskStoplossQueue := NewAskPQueue(1)
	testBidStoplossQueue := NewBidPQueue(0)
	mockDepth := mocks.NewMockMarketDepthStream(mockControl)

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
		depth:       mockDepth,
	}

	return mockControl, ob, &testAskQueue, &testBidQueue, &testAskStoplossQueue, &testBidStoplossQueue, mockDepth, stockID, stockQuantity, stockPrice
}

func TestLoadOldAsk(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, mockAskQueue, _, _, _, mockDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitAsk := makeAsk(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockAskQueue.EXPECT().Push(limitAsk).Times(1)
	mockDepth.EXPECT().AddOrder(false, true, stockPrice, stockQuantity).Times(1)

	ob.LoadOldAsk(limitAsk)

}

func TestLoadOldAskStopLoss(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, mockAskStoplossQueue, _, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	stopLossAsk := makeAsk(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")

	mockAskStoplossQueue.EXPECT().Push(stopLossAsk).Times(1)

	ob.LoadOldAsk(stopLossAsk)

}

func TestLoadOldBid(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, mockBidQueue, _, _, mockDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitBid := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockBidQueue.EXPECT().Push(limitBid).Times(1)
	mockDepth.EXPECT().AddOrder(false, false, stockPrice, stockQuantity).Times(1)

	ob.LoadOldBid(limitBid)

}

func TestLoadOldBidStopLoss(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, mockBidStoplossQueue, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	stopLossBid := makeBid(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")

	mockBidStoplossQueue.EXPECT().Push(stopLossBid).Times(1)

	ob.LoadOldBid(stopLossBid)

}

func TestLoadOldTransactions(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, mockDepth, stockID, _, _ := helperForOrderBookTests(t)
	defer mockControl.Finish()

	t1 := &models.Transaction{
		Id:            1,
		UserId:        10,
		StockId:       stockID,
		Type:          1,
		StockQuantity: 20,
		Price:         50,
		Total:         1000,
		CreatedAt:     "random1",
	}

	t2 := &models.Transaction{
		Id:            2,
		UserId:        10,
		StockId:       stockID,
		Type:          2,
		StockQuantity: 10,
		Price:         20,
		Total:         200,
		CreatedAt:     "random2",
	}

	transactions := []*models.Transaction{t1, t2}

	for _, transaction := range transactions {
		mockDepth.EXPECT().AddTrade(transaction.Price, uint64(-transaction.StockQuantity), transaction.CreatedAt).Times(1)
	}

	ob.LoadOldTransactions(transactions)
}

func TestAddAskToDepth(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, mockDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitAsk := makeAsk(1, stockID, models.Limit, stockQuantity, stockPrice, "")
	askUnfulfilledQuantity := limitAsk.StockQuantity - limitAsk.StockQuantityFulfilled

	mockDepth.EXPECT().AddOrder(isMarket(limitAsk.OrderType), true, limitAsk.Price, askUnfulfilledQuantity)

	ob.addAskToDepth(limitAsk)
}

func TestAddBidToDepth(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, mockDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitBid := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")
	bidUnfulfilledQuantity := limitBid.StockQuantity - limitBid.StockQuantityFulfilled

	mockDepth.EXPECT().AddOrder(isMarket(limitBid.OrderType), false, limitBid.Price, bidUnfulfilledQuantity)

	ob.addBidToDepth(limitBid)
}

func TestOrderBookAddAskOrder(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitAsk1 := makeAsk(1, stockID, models.Limit, stockQuantity, stockPrice, "")
	limitAsk2 := makeAsk(1, stockID+1, models.Limit, stockQuantity+10, stockPrice+20, "")

	go ob.AddAskOrder(limitAsk1)
	time.Sleep(time.Second)
	go ob.AddAskOrder(limitAsk2)

	recAsk1, ok := <-ob.askChan
	if !ok || recAsk1 != limitAsk1 {
		t.Fail()
	}

	recAsk2, ok := <-ob.askChan
	if !ok || recAsk2 != limitAsk2 {
		t.Fail()
	}

}

func TestOrderBookAddAskOrderStoploss(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, mockAskStoplossQueue, _, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	stopLossAsk := makeAsk(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")
	mockAskStoplossQueue.EXPECT().Push(stopLossAsk).Times(1)

	ob.AddAskOrder(stopLossAsk)
}

func TestOrderBookAddBidOrder(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitBid1 := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")
	limitBid2 := makeBid(1, stockID+1, models.Limit, stockQuantity+10, stockPrice+20, "")

	go ob.AddBidOrder(limitBid1)
	time.Sleep(time.Second)

	go ob.AddBidOrder(limitBid2)

	recBid1, ok := <-ob.bidChan
	if !ok || recBid1 != limitBid1 {
		t.Fail()
	}

	recBid2, ok := <-ob.bidChan
	if !ok || recBid2 != limitBid2 {
		t.Fail()
	}

}

func TestOrderBookAddBidOrderStoploss(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, mockBidStoplossQueue, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	stopLossBid := makeBid(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")
	mockBidStoplossQueue.EXPECT().Push(stopLossBid).Times(1)

	ob.AddBidOrder(stopLossBid)
}

func TestOrderBookCancelAskOrder(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, mockDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitAsk := makeAsk(1, stockID, models.Limit, stockQuantity, stockPrice, "")
	unfulfilled := limitAsk.StockQuantity - limitAsk.StockQuantityFulfilled

	mockDepth.EXPECT().CloseOrder(isMarket(limitAsk.OrderType), true, limitAsk.Price, unfulfilled)

	ob.CancelAskOrder(limitAsk)

}

func TestOrderBookCancelAskOrderStoploss(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	stoplossAsk := makeAsk(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")

	ob.CancelAskOrder(stoplossAsk)

}

func TestOrderBookCancelBidOrder(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, mockDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitBid := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")
	unfulfilled := limitBid.StockQuantity - limitBid.StockQuantityFulfilled

	mockDepth.EXPECT().CloseOrder(isMarket(limitBid.OrderType), false, limitBid.Price, unfulfilled)

	ob.CancelBidOrder(limitBid)
}

func TestOrderBookCancelBidOrderStoploss(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, _, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	stoplossBid := makeBid(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")

	ob.CancelBidOrder(stoplossBid)

}

//TestStartStockMatching has tests for both StartStockMatching and clearExistingOrders
func TestStartStockMatching(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, mockAskQueue, mockBidQueue, _, _, mockMarketDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	limitBid := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")
	limitAsk := makeAsk(3, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockBidQueue.EXPECT().Head().AnyTimes()
	mockBidQueue.EXPECT().Size().AnyTimes()
	mockBidQueue.EXPECT().Push(limitBid)

	mockAskQueue.EXPECT().Head().AnyTimes()
	mockAskQueue.EXPECT().Size().AnyTimes()
	mockAskQueue.EXPECT().Push(limitAsk)

	mockMarketDepth.EXPECT().AddOrder(false, true, stockPrice, stockQuantity).Times(1)
	mockMarketDepth.EXPECT().AddOrder(false, false, stockPrice, stockQuantity).Times(1)

	ob.StartStockMatching()
	go ob.AddAskOrder(limitAsk)
	go ob.AddBidOrder(limitBid)

	time.Sleep(time.Second * 5)

	if mockAskQueue.Head() != nil || mockBidQueue.Head() != nil {
		fmt.Println(mockAskQueue.Head())
	}

}

func TestTopMatchingAsk(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, mockMarketDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTestsWithoutMockPQ(t)
	defer mockControl.Finish()

	marketBid := makeBid(1, stockID, models.Market, stockQuantity, stockPrice, "")
	marketAsk := makeAsk(3, stockID, models.Market, stockQuantity, stockPrice, "")

	mockMarketDepth.EXPECT().AddOrder(true, true, stockPrice, stockQuantity).Times(1)
	go ob.StartStockMatching()
	go ob.AddAskOrder(marketAsk)

	time.Sleep(time.Second * 2)

	topAsk, _ := ob.getTopMatchingAsk(marketBid)

	if topAsk != marketAsk {
		t.Errorf("%v %v", topAsk, marketAsk)
	}

}

func TestTopMatchingBid(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, _, _, mockMarketDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTestsWithoutMockPQ(t)
	defer mockControl.Finish()

	marketBid := makeBid(1, stockID, models.Market, stockQuantity, stockPrice, "")
	marketAsk := makeAsk(3, stockID, models.Market, stockQuantity, stockPrice, "")

	mockMarketDepth.EXPECT().AddOrder(true, false, stockPrice, stockQuantity).Times(1)

	go ob.StartStockMatching()
	go ob.AddBidOrder(marketBid)

	time.Sleep(time.Second * 2)

	topBid, _ := ob.getTopMatchingBid(marketAsk)

	if topBid != marketBid {
		t.Errorf("Did not return top matching Bid.")
	}

}

func TestProcessAsk(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, mockAskQueue, mockBidQueue, _, _, mockMarketDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	marketBid := makeBid(1, stockID, models.Market, stockQuantity, stockPrice, "")
	marketAsk := makeAsk(3, stockID, models.Market, stockQuantity, stockPrice, "")

	mockBidQueue.EXPECT().Head().AnyTimes()
	mockBidQueue.EXPECT().Size().AnyTimes()

	mockAskQueue.EXPECT().Head().AnyTimes()
	mockAskQueue.EXPECT().Size().AnyTimes()
	mockAskQueue.EXPECT().Push(marketAsk)

	mockMarketDepth.EXPECT().AddOrder(true, true, stockPrice, stockQuantity).Times(1)

	go ob.AddAskOrder(marketAsk)
	go ob.AddBidOrder(marketBid)
	time.Sleep(time.Second)

	go ob.processAsk(marketAsk)
	time.Sleep(time.Second * 2)

	if mockAskQueue.Head() != nil || mockBidQueue.Head() != nil {
		t.Errorf("Ask was not processed")
	}

}

func TestProcessBid(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, mockAskQueue, mockBidQueue, _, _, mockMarketDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	marketBid := makeBid(1, stockID, models.Market, stockQuantity, stockPrice, "")
	marketAsk := makeAsk(3, stockID, models.Market, stockQuantity, stockPrice, "")

	mockBidQueue.EXPECT().Head().AnyTimes()
	mockBidQueue.EXPECT().Size().AnyTimes()
	mockBidQueue.EXPECT().Push(marketBid)

	mockAskQueue.EXPECT().Head().AnyTimes()
	mockAskQueue.EXPECT().Size().AnyTimes()

	mockMarketDepth.EXPECT().AddOrder(true, false, stockPrice, stockQuantity).Times(1)

	go ob.AddAskOrder(marketAsk)
	time.Sleep(time.Second * 1)
	go ob.AddBidOrder(marketBid)
	time.Sleep(time.Second * 2)

	ob.processBid(marketBid)

	if mockBidQueue.Head() != nil || mockAskQueue.Head() != nil {
		t.Errorf("Bid was not processed")
	}

}

func TestTriggerStoplosses(t *testing.T) {
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, _, _, mockAskStoplossQueue, mockBidStoplossQueue, _, stockID, stockQuantity, _ := helperForOrderBookTests(t)
	defer mockControl.Finish()

	t1 := &models.Transaction{
		Id:            1,
		UserId:        10,
		StockId:       stockID,
		Type:          1,
		StockQuantity: 20,
		Price:         50,
		Total:         1000,
		CreatedAt:     "random1",
	}

	stoplossBid := makeBid(1, stockID, models.StopLoss, stockQuantity, 30, "")
	stoplossAsk := makeAsk(3, stockID, models.StopLoss, stockQuantity, 70, "")

	mockBidStoplossQueue.EXPECT().Head().AnyTimes()
	mockBidStoplossQueue.EXPECT().Size().AnyTimes()
	mockBidStoplossQueue.EXPECT().Push(stoplossBid)

	mockAskStoplossQueue.EXPECT().Head().AnyTimes()
	mockAskStoplossQueue.EXPECT().Size().AnyTimes()
	mockAskStoplossQueue.EXPECT().Push(stoplossAsk)

	go ob.AddAskOrder(stoplossAsk)
	go ob.AddBidOrder(stoplossBid)
	time.Sleep(time.Second)
	go ob.triggerStopLosses(t1)
	time.Sleep(time.Second * 2)

	if ob.askStoploss.Head() != nil || ob.bidStoploss.Head() != nil {
		t.Errorf("Stoploss was not triggered")
	}
}

func TestWaitForOrders(t *testing.T) {
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl, ob, mockAskQueue, mockBidQueue, _, _, mockMarketDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)
	defer mockControl.Finish()

	marketBid := makeBid(1, stockID, models.Market, stockQuantity, stockPrice, "")
	marketAsk := makeAsk(3, stockID, models.Market, stockQuantity, stockPrice, "")

	mockBidQueue.EXPECT().Head().AnyTimes()
	mockBidQueue.EXPECT().Size().AnyTimes()

	mockAskQueue.EXPECT().Head().AnyTimes()
	mockAskQueue.EXPECT().Size().AnyTimes()
	mockAskQueue.EXPECT().Push(marketAsk)

	mockMarketDepth.EXPECT().AddOrder(true, true, stockPrice, stockQuantity).Times(1)

	go ob.waitForOrder()
	time.Sleep(time.Second)
	go ob.AddAskOrder(marketAsk)
	time.Sleep(time.Second)
	go ob.AddBidOrder(marketBid)
	time.Sleep(time.Second)

	if ob.asks.Head() != nil || ob.bids.Head() != nil {
		t.Errorf("Wait for order failed")
	}
}

// func Test_MakeTrade(t *testing.T) { // to be completed
// 	ob, _, _, _, _, mockDepth, stockID, stockQuantity, stockPrice := helperForOrderBookTests(t)

// 	marketBid := makeBid(1, stockID, models.Market, stockQuantity, stockPrice, "")
// 	marketAsk := makeAsk(3, stockID, models.Market, stockQuantity, stockPrice, "")

// 	go ob.AddAskOrder(marketAsk)
// 	go ob.AddBidOrder(marketBid)
// 	time.Sleep(time.Second * 2)

// 	mockDepth.EXPECT().CloseOrder(isMarket(marketBid.OrderType), false, marketBid.Price, uint32(-stockQuantity))
// 	mockDepth.EXPECT().CloseOrder(isMarket(marketAsk.OrderType), false, marketAsk.Price, uint32(-stockQuantity))
// 	mockDepth.EXPECT().AddTrade(gomock.Any, gomock.Any, gomock.Any)

// askStatus, bidStatus := ob.makeTrade(marketAsk, marketBid, false, false)

// 	if !askStatus || !bidStatus {
// 		t.Errorf("Trade not done")
// 	}
// }

func Test_MakeTrade(t *testing.T) {

	config := utils.GetConfiguration()
	utils.Init(config)

	var l = utils.Logger.WithFields(logrus.Fields{})

	var stockID uint32 = 1
	// var stockQuantity uint32 = 10
	// var stockPrice uint32 = 20
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

	ob.LoadOldAsk(ask)
	ob.LoadOldBid(bid)
	time.Sleep(time.Second * 2)

	askStatus, bidStatus := ob.makeTrade(ask, bid, false, false)

	if !askStatus || !bidStatus {
		l.Errorf("Errored in testMakeTrade")
	}

}
