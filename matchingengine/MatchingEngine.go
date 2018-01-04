package matchingengine

import (
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
func NewMatchingEngine(dsm datastreams.Manager) MatchingEngine {
	engine := &matchingEngine{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module": "matchingengine",
		}),
		orderBooks:         make(map[uint32]OrderBook),
		datastreamsManager: dsm,
	}

	engine.loadOldOrders()

	for _, ob := range engine.orderBooks {
		go ob.StartStockMatching()
	}

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

// loadOldOrders() loads old unfulfilled orders from database
func (m *matchingEngine) loadOldOrders() {
	var l = m.logger.WithFields(logrus.Fields{
		"method": "loadOldOrders",
	})

	db, err := models.DbOpen()
	if err != nil {
		l.Errorf("Errored : %+v", err)
		panic("Error opening database for matching engine")
	}
	defer db.Close()

	var (
		openAskOrders []*models.Ask
		openBidOrders []*models.Bid
		stockIds      []uint32
	)

	//Load stock ids from database
	if err := db.Model(&models.Stock{}).Pluck("id", &stockIds).Error; err != nil {
		panic("Failed to load stock ids in matching engine: " + err.Error())
	}

	//Load open ask orders from database
	if err := db.Where("isClosed = ?", 0).Find(&openAskOrders).Error; err != nil {
		panic("Error loading open ask orders in matching engine: " + err.Error())
	}

	//Load open bid orders from database
	if err := db.Where("isClosed = ?", 0).Find(&openBidOrders).Error; err != nil {
		panic("Error loading open bid orders in matching engine: " + err.Error())
	}

	for _, stockId := range stockIds {
		marketDepth := m.datastreamsManager.GetMarketDepthStream(stockId)
		m.orderBooks[stockId] = NewOrderBook(stockId, marketDepth)
	}

	//Load open ask orders into priority queue
	for _, openAskOrder := range openAskOrders {
		m.orderBooks[openAskOrder.StockId].AddAskOrder(openAskOrder)
	}

	//Load open bid orders into priority queue
	for _, openBidOrder := range openBidOrders {
		m.orderBooks[openBidOrder.StockId].AddBidOrder(openBidOrder)
	}
}
