package matchingengine

import (
	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// FillOrder is a type definition for a function that fills an order with given ask, bid, stockPrice and stockQty
type FillOrder func(ask *models.Ask, bid *models.Bid, stockTradePrice uint32, stockTradeQty uint32) (askDone bool, bidDone bool, tr *models.Transaction)

// fillOrderFn is the actual function that filles an order.
// It has been separated from implementation to ease testing.
var fillOrderFn FillOrder = models.PerformOrderFillTransaction

// OrderBook stores the order book for a given stock
type OrderBook interface {
	AddAskOrder(*models.Ask)
	AddBidOrder(*models.Bid)
	StartStockMatching()
}

// orderBook implements the OrderBook interface
type orderBook struct {
	stockId     uint32
	askChan     chan *models.Ask
	bidChan     chan *models.Bid
	asks        *AskPQueue
	bids        *BidPQueue
	askStoploss *AskPQueue
	bidStoploss *BidPQueue
	depth       *datastreams.MarketDepth
}

// NewOrderBook returns a new OrderBook instance for a given stockId.
func NewOrderBook(stockId uint32) OrderBook {
	return &orderBook{
		stockId:     stockId,
		askChan:     make(chan *models.Ask),
		bidChan:     make(chan *models.Bid),
		asks:        NewAskPQueue(MINPQ), //lower price has higher priority
		bids:        NewBidPQueue(MAXPQ), //higher price has higher priority
		askStoploss: NewAskPQueue(MAXPQ), // stoplosses work like opposite of limit/market.
		bidStoploss: NewBidPQueue(MINPQ), // They sell when price goes below a certain trigger price.
		depth:       datastreams.NewMarketDepth(stockId),
	}
}

// AddAskOrder adds an ask order to the book. It will take care of adding stopLoss, or partially filled orders automatically
func (ob *orderBook) AddAskOrder(ask *models.Ask) {
	if ask.IsClosed {
		return
	}

	if ask.OrderType == models.StopLoss {
		ob.askStoploss.Push(ask, ask.Price, 0)
	} else {
		bidUnfulfilledQuantity := ask.StockQuantity - ask.StockQuantityFulfilled
		ob.asks.Push(ask, ask.Price, bidUnfulfilledQuantity)
	}
}

// AddBidOrder adds an bid order to the book. It will take care of adding stopLoss, or partially filled orders automatically
func (ob *orderBook) AddBidOrder(bid *models.Bid) {
	if bid.IsClosed {
		return
	}

	if bid.OrderType == models.StopLoss {
		ob.bidStoploss.Push(bid, bid.Price, 0)
	} else {
		bidUnfulfilledQuantity := bid.StockQuantity - bid.StockQuantityFulfilled
		ob.bids.Push(bid, bid.Price, bidUnfulfilledQuantity)
	}
}

/*
 *	StartStockMatching listens for incoming orders for a particular stock and process them
 */
func (ob *orderBook) StartStockMatching() {
	var l = logger.WithFields(logrus.Fields{
		"method":  "startStockMatching",
		"stockId": ob.stockId,
	})

	l.Infof("Started with ask_count = %d, bid_count = %d", ob.asks.Size(), ob.bids.Size())

	// Clear the existing orders first
	ob.processOrder()

	// run infinite loop
	for {
		ob.waitForOrder()
		ob.processOrder()
	}
}

/*
 *	Method to check and trigger(if possible) StopLoss orders whenever a transaction occurs
 */
func (ob *orderBook) triggerStopLosses(tr *models.Transaction) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "triggerStopLosses",
		"stockId": ob.stockId,
	})

	db, err := utils.DbOpen()
	if err != nil {
		l.Errorf("Error while opening DB to trigger stoplosses. Not triggering them.")
		return
	}
	defer db.Close()

	l.Debugf("Triggering ask stoplosses")
	topAskStoploss := ob.askStoploss.Head()
	// We trigger asks having higher than current price. That's because if it's an ask stoploss, then we will sell
	// if the current price goes below the trigger price.
	for topAskStoploss != nil && tr.Price <= topAskStoploss.Price {
		l.Debugf("Triggering ask %+v", topAskStoploss)

		topAskStoploss = ob.askStoploss.Pop()
		topAskStoploss.OrderType = models.StopLossActive
		if err := db.Save(topAskStoploss).Error; err != nil {
			l.Errorf("Error while changing state of %+v from StopLoss to StopLossActive", topAskStoploss)
		}

		ob.asks.Push(topAskStoploss, topAskStoploss.Price, topAskStoploss.StockQuantity)
		ob.depth.AddOrder(true, true, topAskStoploss.Price, topAskStoploss.StockQuantity)
		topAskStoploss = ob.askStoploss.Head()
	}

	l.Debugf("Triggering bid stoplosses")
	// We trigger bids having lower than current price. That's because if it's a bid stoploss, we will buy if the
	// current price goes above the trigger price.
	topBidStoploss := ob.bidStoploss.Head()
	for topBidStoploss != nil && tr.Price >= topBidStoploss.Price {
		l.Debugf("Triggering bid %+v", topBidStoploss)

		topBidStoploss = ob.bidStoploss.Pop()
		topBidStoploss.OrderType = models.StopLossActive
		if err := db.Save(topBidStoploss).Error; err != nil {
			l.Errorf("Error while changing state of %+v from StopLoss to StopLossActive", topBidStoploss)
		}

		ob.bids.Push(topBidStoploss, topBidStoploss.Price, topBidStoploss.StockQuantity)
		ob.depth.AddOrder(true, false, topBidStoploss.Price, topBidStoploss.StockQuantity)
		topBidStoploss = ob.bidStoploss.Head()
	}
}

/*
 *	Method to return a matching ask and bid belonging to distinct users
 */
func (ob *orderBook) getTopMatchingOrders() (*models.Ask, *models.Bid, func()) {
	askTop, bidTop := ob.asks.Head(), ob.bids.Head()

	// If either one is nil, there are no matching orders
	if askTop == nil || bidTop == nil {
		return askTop, bidTop, func() {}
	}

	// Store the same user's bids in a slice
	var sameUserBids []*models.Bid
	for bidTop != nil && bidTop.UserId == askTop.UserId {
		sameUserBids = append(sameUserBids, ob.bids.Pop())
		bidTop = ob.bids.Head()
	}

	// Function to add back the sameUserBids in the PQueue
	addBackOrders := func() {
		for _, bid := range sameUserBids {
			ob.bids.Push(bid, bid.Price, bid.StockQuantity)
		}
	}

	// If orders don't match, push back the same user bids and return
	if bidTop == nil || !isOrderMatching(askTop, bidTop) {
		addBackOrders()
		return nil, nil, func() {}
	}

	// If control reaches here, we return a non-nil matching pair of orders
	return askTop, bidTop, addBackOrders
}

/*
 *	Method to process the orders for a particular stock and carry out transactions
 */
func (ob *orderBook) processOrder() {
	var l = logger.WithFields(logrus.Fields{
		"method":  "processOrder",
		"stockId": ob.stockId,
	})

	var (
		askDone bool
		bidDone bool
		tr      *models.Transaction
	)

	askTop, bidTop, addBackOrders := ob.getTopMatchingOrders()

	for bidTop != nil && askTop != nil {

		l.Debugf("Performing OrderFill transaction")

		/*
		 *	fillOrderFn should
		 *		- acquire locks on the users
		 *		- update StockQuantityFulfilled and IsClosed
		 *		- record the transaction in the database
		 */
		stockTradePrice, stockTradeQty := getTradePriceAndQty(askTop, bidTop)
		askDone, bidDone, tr = fillOrderFn(askTop, bidTop, stockTradePrice, stockTradeQty)

		if tr != nil {
			l.Infof("Trade made between ask_id %d and bid %d", askTop.Id, bidTop.Id)
			// tr is always AskTransaction. So its StockQty < 0. Make it positive.
			ob.depth.Trade(tr.Price, uint32(-tr.StockQuantity), tr.CreatedAt)

			ob.depth.CloseOrder(isMarket(askTop.OrderType), true, askTop.Price, uint32(-tr.StockQuantity))
			ob.depth.CloseOrder(isMarket(bidTop.OrderType), false, bidTop.Price, uint32(-tr.StockQuantity))

			// Trigger stop losses here
			ob.triggerStopLosses(tr)
		}

		if askDone {
			// If transaction didn't happen, but askDone is true
			// Thus the ask was faulty in some way or the order was cancelled
			// Thus, the remaining stock quantity needs to be closed
			if tr == nil {
				ob.depth.CloseOrder(isMarket(askTop.OrderType), true, askTop.Price, askTop.StockQuantity-askTop.StockQuantityFulfilled)
			}
			ob.asks.Pop()
		}
		if bidDone {
			// If transaction didn't happen, but bidDone is true
			// Thus the bid was faulty in some way or the order was cancelled
			// Thus, the remaining stock quantity needs to be closed
			if tr == nil {
				ob.depth.CloseOrder(isMarket(bidTop.OrderType), false, bidTop.Price, bidTop.StockQuantity-bidTop.StockQuantityFulfilled)
			}
			ob.bids.Pop()
		}

		addBackOrders()

		// Check if error occurred in acquiring locks or database transactions
		if askDone == false && bidDone == false {
			l.Errorf("PerformOrderFillTransaction returned both askDone, bidDone false")
			return
		}

		askTop, bidTop, addBackOrders = ob.getTopMatchingOrders()
	}
}

/*
 *	Method to wait for an incoming order on the channels
 */
func (ob *orderBook) waitForOrder() {
	var l = logger.WithFields(logrus.Fields{
		"method": "waitForOrder",
	})

	select {
	case askOrder := <-ob.askChan:
		l.Debugf("Got ask %+v. Processing", askOrder)
		if askOrder.OrderType == models.StopLoss {
			l.Debugf("Adding stopLoss with ask_id %d to the list", askOrder.Id)
			ob.askStoploss.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
			break
		}

		// If control reaches here, it's not a StopLoss order
		ob.asks.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
		ob.depth.AddOrder(isMarket(askOrder.OrderType), true, askOrder.Price, askOrder.StockQuantity)

	case bidOrder := <-ob.bidChan:
		l.Debugf("Got bid %+v. Processing", bidOrder)
		if bidOrder.OrderType == models.StopLoss {
			l.Debugf("Adding stopLoss with bid_id %d to the list", bidOrder.Id)
			ob.bidStoploss.Push(bidOrder, bidOrder.Price, bidOrder.StockQuantity)
			break
		}

		// If control reaches here, it's not a StopLoss order
		ob.bids.Push(bidOrder, bidOrder.Price, bidOrder.StockQuantity)
		ob.depth.AddOrder(isMarket(bidOrder.OrderType), true, bidOrder.Price, bidOrder.StockQuantity)
	}
}
