package matchingengine

import (
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/mocks"
	"github.com/delta/dalal-street-server/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/golang/mock/gomock"
)

// helper function to create a new transaction object
func makeTransaction(id uint32, userID uint32, stockID uint32, transactionType models.TransactionType, stockQty int64, price uint64, total int64, createdAt string) *models.Transaction {
	return &models.Transaction{
		Id:            id,
		UserId:        userID,
		StockId:       stockID,
		Type:          transactionType,
		StockQuantity: stockQty,
		Price:         price,
		Total:         total,
		CreatedAt:     createdAt,
	}
}

func TestLoadOldAsk(t *testing.T) {

	// This initializes the logger. This is required because LoadOldAsk tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl := gomock.NewController(t)
	defer mockControl.Finish()

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

	limitAsk := makeAsk(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockAskQueue.EXPECT().Push(limitAsk).Times(1)
	mockDepth.EXPECT().AddOrder(false, true, stockPrice, stockQuantity).Times(1)

	ob.LoadOldAsk(limitAsk)

}

func TestLoadOldAskStopLoss(t *testing.T) {

	// This initializes the logger. This is required because LoadOldAsk tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl := gomock.NewController(t)
	defer mockControl.Finish()

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

	stopLossAsk := makeAsk(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")

	mockAskStoplossQueue.EXPECT().Push(stopLossAsk).Times(1)

	ob.LoadOldAsk(stopLossAsk)

}

func TestLoadOldBid(t *testing.T) {

	// This initializes the logger. This is required because LoadOldBid tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl := gomock.NewController(t)
	defer mockControl.Finish()

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

	limitBid := makeBid(1, stockID, models.Limit, stockQuantity, stockPrice, "")

	mockBidQueue.EXPECT().Push(limitBid).Times(1)
	mockDepth.EXPECT().AddOrder(false, false, stockPrice, stockQuantity).Times(1)

	ob.LoadOldBid(limitBid)

}

func TestLoadOldBidStopLoss(t *testing.T) {

	// This initializes the logger. This is required because LoadOldAsk tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl := gomock.NewController(t)
	defer mockControl.Finish()

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

	stopLossBid := makeBid(1, stockID, models.StopLoss, stockQuantity, stockPrice, "")

	mockBidStoplossQueue.EXPECT().Push(stopLossBid).Times(1)

	ob.LoadOldBid(stopLossBid)

}

func TestLoadOldTransactions(t *testing.T) {

	// This initializes the logger. This is required because LoadOldAsk tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl := gomock.NewController(t)
	defer mockControl.Finish()

	var stockID uint32 = 1

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

	t1 := makeTransaction(1, 10, stockID, 1, 20, 50, 1000, "random1")
	t2 := makeTransaction(2, 10, stockID, 2, 10, 20, 200, "random2")

	transactions := []*models.Transaction{t1, t2}

	for _, transaction := range transactions {
		mockDepth.EXPECT().AddTrade(transaction.Price, uint64(-transaction.StockQuantity), transaction.CreatedAt).Times(1)
	}

	ob.LoadOldTransactions(transactions)
}
