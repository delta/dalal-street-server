package matchingengine

import (
	"sync"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var logger *logrus.Entry

// MatchingEngine represents a collection of OrderBooks for all stocks in the exchange.
type MatchingEngine interface {
	AddAskOrder(*models.Ask)
	AddBidOrder(*models.Bid)
	CancelAskOrder(*models.Ask)
	CancelBidOrder(*models.Bid)
}

// matchingEngine implements the MatchingEngine interface
type matchingEngine struct {
	logger *logrus.Entry

	// orderBooks stores details of placed orders.
	// Each entry in orderBooks corresponds to a particular stock.
	orderBooks map[uint32]OrderBook

	// datastreamsManager is used to manage datastreams
	datastreamsManager datastreams.Manager
}

// Init configures the matching engine
func Init(config *utils.Config) {
	// nothing to do here
}

// NewMatchingEngine returns an instance of MatchingEngine
// It calls StartStockmatching for all the stocks in concurrent goroutines.
// WARNING: Do NOT call this again for a given stock, once the server has restarted
func NewMatchingEngine(dsm datastreams.Manager) MatchingEngine {
	engine := &matchingEngine{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "matchingengine",
		}),
		orderBooks:         make(map[uint32]OrderBook),
		datastreamsManager: dsm,
	}

	engine.loadOldOrders()

	var wg sync.WaitGroup

	for _, ob := range engine.orderBooks {
		wg.Add(1)
		go func(ob OrderBook) {
			ob.StartStockMatching() // this will return when it's initialized
			wg.Done()
		}(ob)
	}

	wg.Wait() // Don't return till the orderbooks have been initialized
	engine.logger.Info("Started matching engine")
	return engine
}

// AddAskOrder adds an ask order to the relevant order book
func (m *matchingEngine) AddAskOrder(askOrder *models.Ask) {
	m.orderBooks[askOrder.StockId].AddAskOrder(askOrder)
}

// AddBidOrder adds a bid order to the relevant order book
func (m *matchingEngine) AddBidOrder(bidOrder *models.Bid) {
	m.orderBooks[bidOrder.StockId].AddBidOrder(bidOrder)
}

// CancelAskOrder removes the ask order from the orderbook.
func (m *matchingEngine) CancelAskOrder(askOrder *models.Ask) {
	m.orderBooks[askOrder.StockId].CancelAskOrder(askOrder)
}

// CancelBidOrder removes the bid order from the orderbook.
func (m *matchingEngine) CancelBidOrder(bidOrder *models.Bid) {
	m.orderBooks[bidOrder.StockId].CancelBidOrder(bidOrder)
}

// loadOldOrders() loads old unfulfilled orders from database
func (m *matchingEngine) loadOldOrders() {
	var l = m.logger.WithFields(logrus.Fields{
		"method": "loadOldOrders",
	})

	db := utils.GetDB()

	var (
		openAskOrders []*models.Ask
		openBidOrders []*models.Bid
		stockIDs      []uint32
		err           error
	)

	//Load stock ids from database
	if err = db.Model(&models.Stock{}).Pluck("id", &stockIDs).Error; err != nil {
		panic("Failed to load stock ids in matching engine: " + err.Error())
	}

	//Load open ask orders from database
	openAskOrders, err = models.GetAllOpenAsks()
	if err != nil {
		panic("Error loading open ask orders in matching engine: " + err.Error())
	}

	//Load open bid orders from database
	openBidOrders, err = models.GetAllOpenBids()
	if err != nil {
		panic("Error loading open bid orders in matching engine: " + err.Error())
	}

	for _, stockID := range stockIDs {
		marketDepth := m.datastreamsManager.GetMarketDepthStream(stockID)
		m.orderBooks[stockID] = NewOrderBook(stockID, marketDepth)
		tx, err := models.GetAskTransactionsForStock(stockID, 15)
		if err != nil {
			l.Errorf("Unable to load old transactions for stockid %d", stockID)
		} else {
			m.orderBooks[stockID].LoadOldTransactions(tx)
		}
	}

	//Load open ask orders into priority queue
	for _, openAskOrder := range openAskOrders {
		m.orderBooks[openAskOrder.StockId].LoadOldAsk(openAskOrder)
	}

	//Load open bid orders into priority queue
	for _, openBidOrder := range openBidOrders {
		m.orderBooks[openBidOrder.StockId].LoadOldBid(openBidOrder)
	}

	l.Info("Loaded!")
}
