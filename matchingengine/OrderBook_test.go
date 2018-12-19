package matchingengine

import (
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/mocks"
	"github.com/delta/dalal-street-server/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/golang/mock/gomock"
)

func TestLoadOldAsk(t *testing.T) {

	// This initializes the logger. This is required because LoadOldAsk tries accessing the logger
	// and if it is not initialized, a nil reference error is thrown.
	config := utils.GetConfiguration()
	utils.Init(config)

	mockControl := gomock.NewController(t)
	defer mockControl.Finish()

	var stockID uint32 = 1
	var stockQuantity uint32 = 10
	var stockPrice uint32 = 20

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
