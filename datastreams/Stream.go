package datastreams

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry

// listener represents a single listener in the stream
type listener struct {
	update chan interface{}
	done   <-chan struct{}
}

// Init initalizes and configures the datastreams module
func Init(config *utils.Config) {
	logger = utils.GetNewFileLogger("datastreams.log", 20, "debug", false).WithFields(logrus.Fields{
		"module": "datastreams",
	})
}

// Manager manages access to all data streams
type Manager interface {
	GetMarketDepthStream(stockId uint32) MarketDepthStream
	GetMarketEventsStream() MarketEventsStream
	GetMyOrdersStream() MyOrdersStream
	GetNotificationsStream() NotificationsStream
	GetStockExchangeStream() StockExchangeStream
	GetStockPricesStream() StockPricesStream
	GetTransactionsStream() TransactionsStream
}

// dataStreamsManager implements the Manager interface
type dataStreamsManager struct {
	logger *logrus.Entry

	// market depth streams
	marketDepthsLock sync.RWMutex
	marketDepthsMap  map[uint32]MarketDepthStream

	// market events stream
	marketEventsStreamInstance MarketEventsStream
	// myorders stream
	myOrdersStreamInstance MyOrdersStream
	// notification stream
	notificationsStreamInstance NotificationsStream
	// stock exchange stream
	stockExchangeStreamInstance StockExchangeStream
	// stock prices stream
	stockPricesStreamInstance StockPricesStream
	// transactions stream
	transactionsStreamInstance TransactionsStream
}

// dataStreamsManagerInstance holds the singleton instance of dataStreamsManager
var dataStreamsManagerInstance *dataStreamsManager

// GetManager returns a singleton instance of Manager
func GetManager() Manager {
	return &dataStreamsManager{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "datastreams.Manager",
		}),

		marketDepthsMap:             make(map[uint32]MarketDepthStream),
		marketEventsStreamInstance:  newMarketEventsStream(),
		myOrdersStreamInstance:      newMyOrdersStream(),
		notificationsStreamInstance: newNotificationsStream(),
		stockExchangeStreamInstance: newStockExchangeStream(),
		stockPricesStreamInstance:   newStockPricesStream(),
		transactionsStreamInstance:  newTransactionsStream(),
	}
}

// GetMarketDepthStream returns a singleton instance MarketDepthStream for a given stockId
func (dsm *dataStreamsManager) GetMarketDepthStream(stockId uint32) MarketDepthStream {
	dsm.marketDepthsLock.Lock()
	if dsm.marketDepthsMap[stockId] == nil {
		dsm.marketDepthsMap[stockId] = newMarketDepthStream(stockId)
	}
	stream := dsm.marketDepthsMap[stockId]
	dsm.marketDepthsLock.Unlock()

	return stream
}

// GetMarketEventsStream returns a singleton instance of MarketEvents stream
func (dsm *dataStreamsManager) GetMarketEventsStream() MarketEventsStream {
	return dsm.marketEventsStreamInstance
}

// GetMyOrdersStream returns a singleton instance of MyOrders stream
func (dsm *dataStreamsManager) GetMyOrdersStream() MyOrdersStream {
	return dsm.myOrdersStreamInstance
}

// GetNotificationsStream returns a singleton instance of Notifications stream
func (dsm *dataStreamsManager) GetNotificationsStream() NotificationsStream {
	return dsm.notificationsStreamInstance
}

// GetStockExchangeStream returns a singleton instance of StockExchange stream
func (dsm *dataStreamsManager) GetStockExchangeStream() StockExchangeStream {
	return dsm.stockExchangeStreamInstance
}

// GetStockPricesStream returns a singleton instance of StockPrices stream
func (dsm *dataStreamsManager) GetStockPricesStream() StockPricesStream {
	return dsm.stockPricesStreamInstance
}

// GetTransactionsStream returns a singleton instance of Transactions stream
func (dsm *dataStreamsManager) GetTransactionsStream() TransactionsStream {
	return dsm.transactionsStreamInstance
}
