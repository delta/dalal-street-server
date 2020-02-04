package datastreams

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils"
)

//go:generate mockgen -source Stream.go -destination ../mocks/mock_Stream.go -package mocks

var logger *logrus.Entry

// listener represents a single listener in the stream
type listener struct {
	update chan interface{}
	done   <-chan struct{}
}

// Init initializes and configures the datastreams module
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
	GetStockHistoryStream(stockId uint32) StockHistoryStream
	GetGameStateStream() GameStateStream
}

// dataStreamsManager implements the Manager interface
type dataStreamsManager struct {
	logger *logrus.Entry

	// market depth streams
	marketDepthsLock sync.RWMutex
	marketDepthsMap  map[uint32]MarketDepthStream

	// stock history streams
	stockHistoryLock      sync.RWMutex
	stockHistoryStreamMap map[uint32]StockHistoryStream

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
	// game state stream
	gameStateStreamInstance GameStateStream
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
		stockHistoryStreamMap:       make(map[uint32]StockHistoryStream),
		marketEventsStreamInstance:  newMarketEventsStream(),
		myOrdersStreamInstance:      newMyOrdersStream(),
		notificationsStreamInstance: newNotificationsStream(),
		stockExchangeStreamInstance: newStockExchangeStream(),
		stockPricesStreamInstance:   newStockPricesStream(),
		transactionsStreamInstance:  newTransactionsStream(),
		gameStateStreamInstance:     newGameStateStream(),
	}
}

// GetMarketDepthStream returns a singleton instance MarketDepthStream for a given stockId
func (dsm *dataStreamsManager) GetMarketDepthStream(stockId uint32) MarketDepthStream {
	dsm.marketDepthsLock.Lock()
	defer dsm.marketDepthsLock.Unlock()

	_, ok := dsm.marketDepthsMap[stockId]
	if !ok {
		dsm.marketDepthsMap[stockId] = newMarketDepthStream(stockId)
	}
	return dsm.marketDepthsMap[stockId]
}

// GetMarketEventsStream returns a singleton instance of MarketEvents stream
func (dsm *dataStreamsManager) GetMarketEventsStream() MarketEventsStream {
	return dsm.marketEventsStreamInstance
}

// GetStockHistoryStream returns a singleton instance StockHistoryStream for a given stockId
func (dsm *dataStreamsManager) GetStockHistoryStream(stockId uint32) StockHistoryStream {
	dsm.stockHistoryLock.Lock()
	defer dsm.stockHistoryLock.Unlock()

	_, ok := dsm.stockHistoryStreamMap[stockId]
	if !ok {
		dsm.stockHistoryStreamMap[stockId] = newStockHistoryStream(stockId)
	}
	return dsm.stockHistoryStreamMap[stockId]
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

func (dsm *dataStreamsManager) GetGameStateStream() GameStateStream {
	return dsm.gameStateStreamInstance
}
