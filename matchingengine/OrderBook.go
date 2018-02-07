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
	LoadOldAsk(*models.Ask)
	LoadOldBid(*models.Bid)
	AddAskOrder(*models.Ask)
	AddBidOrder(*models.Bid)
	StartStockMatching()
}

// orderBook implements the OrderBook interface
type orderBook struct {
	logger      *logrus.Entry
	stockId     uint32
	askChan     chan *models.Ask
	bidChan     chan *models.Bid
	asks        *AskPQueue
	bids        *BidPQueue
	askStoploss *AskPQueue
	bidStoploss *BidPQueue
	depth       datastreams.MarketDepthStream
}

// NewOrderBook returns a new OrderBook instance for a given stockId.
func NewOrderBook(stockId uint32, mds datastreams.MarketDepthStream) OrderBook {
	return &orderBook{
		logger: utils.Logger.WithFields(logrus.Fields{
			"module":        "matchingengine.OrderBook",
			"param_stockId": stockId,
		}),
		stockId:     stockId,
		askChan:     make(chan *models.Ask),
		bidChan:     make(chan *models.Bid),
		asks:        NewAskPQueue(MINPQ), //lower price has higher priority
		bids:        NewBidPQueue(MAXPQ), //higher price has higher priority
		askStoploss: NewAskPQueue(MAXPQ), // stoplosses work like opposite of limit/market.
		bidStoploss: NewBidPQueue(MINPQ), // They sell when price goes below a certain trigger price.
		depth:       mds,
	}
}

// addAskToQueue adds a Market or Limit ask to the queue
func (ob *orderBook) addAskToQueue(ask *models.Ask) {
	l := ob.logger.WithFields(logrus.Fields{
		"method": "addAskToQueue",
	})

	l.Infof("Adding ask with ask_id = %d to queue", ask.Id)

	askUnfulfilledQuantity := ask.StockQuantity - ask.StockQuantityFulfilled
	ob.asks.Push(ask, ask.Price, askUnfulfilledQuantity)
	ob.depth.AddOrder(isMarket(ask.OrderType), true, ask.Price, ask.StockQuantity)
}

// addBidToQueue adds a Market or Limit bid to the queue
func (ob *orderBook) addBidToQueue(bid *models.Bid) {
	l := ob.logger.WithFields(logrus.Fields{
		"method": "addBidToQueue",
	})

	l.Infof("Adding bid with bid_id = %d to queue", bid.Id)

	bidUnfulfilledQuantity := bid.StockQuantity - bid.StockQuantityFulfilled
	ob.bids.Push(bid, bid.Price, bidUnfulfilledQuantity)
	ob.depth.AddOrder(isMarket(bid.OrderType), false, bid.Price, bid.StockQuantity)
}

func (ob *orderBook) LoadOldAsk(ask *models.Ask) {
	ob.addAskToQueue(ask)
}

func (ob *orderBook) LoadOldBid(bid *models.Bid) {
	ob.addBidToQueue(bid)
}

// AddAskOrder adds an ask order to the book. It will take care of adding stopLoss, or partially filled orders automatically
func (ob *orderBook) AddAskOrder(ask *models.Ask) {
	ob.askChan <- ask
}

// AddBidOrder adds an bid order to the book. It will take care of adding stopLoss, or partially filled orders automatically
func (ob *orderBook) AddBidOrder(bid *models.Bid) {
	ob.bidChan <- bid
}

/*
 *	StartStockMatching listens for incoming orders for a particular stock and process them
 */
func (ob *orderBook) StartStockMatching() {
	var l = ob.logger.WithFields(logrus.Fields{
		"method": "startStockMatching",
	})

	l.Infof("Started with ask_count = %d, bid_count = %d", ob.asks.Size(), ob.bids.Size())

	// Clear the existing orders first
	ob.clearExistingOrders()

	// run infinite loop
	for {
		ob.waitForOrder()
	}
}

// getTopMatchingBid checks for a matching bid from the orderbook for an incoming ask
func (ob *orderBook) getTopMatchingBid(ask *models.Ask) (*models.Bid, func()) {
	bidTop := ob.bids.Head()

	// Store the same user's bids in a slice
	var sameUserBids []*models.Bid
	for bidTop != nil && bidTop.UserId == ask.UserId {
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
	if bidTop == nil || !isOrderMatching(ask, bidTop) {
		addBackOrders()
		return nil, func() {}
	}

	// If control reaches here, we return a non-nil matching bid
	return bidTop, addBackOrders
}

// getTopMatchingAsk returns a matching ask from the orderbook for an incomming bid
func (ob *orderBook) getTopMatchingAsk(bid *models.Bid) (*models.Ask, func()) {
	askTop := ob.asks.Head()

	// Store the same user's asks in a slice
	var sameUserAsks []*models.Ask
	for askTop != nil && askTop.UserId == bid.UserId {
		sameUserAsks = append(sameUserAsks, ob.asks.Pop())
		askTop = ob.asks.Head()
	}

	// Function to add back the sameUserAsks in the PQueue
	addBackOrders := func() {
		for _, ask := range sameUserAsks {
			ob.asks.Push(ask, ask.Price, ask.StockQuantity)
		}
	}

	// If orders don't match, push back the same user asks and return
	if askTop == nil || !isOrderMatching(askTop, bid) {
		addBackOrders()
		return nil, func() {}
	}

	// If control reaches here, we return a non-nil matching ask
	return askTop, addBackOrders
}

// processAsk tries to match an incoming ask with existing bids
// and carry out a trade if possible
func (ob *orderBook) processAsk(ask *models.Ask) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method":   "processAsk",
		"paramAsk": ask,
	})

	// in case of stoploss order, add it to stoploss queue and return
	if ask.OrderType == models.StopLoss {
		l.Debugf("Adding stopLoss with ask_id %d to the queue", ask.Id)
		ob.askStoploss.Push(ask, ask.Price, ask.StockQuantity)
		return
	}

	// if control reaches here, it's NOT a stoploss order
	var askDone, bidDone bool
	matchingBid, addBackOrders := ob.getTopMatchingBid(ask)

	for matchingBid != nil {
		/*
		 * makeTrade invokes fillOrderFn
		 * It handles changes in market depth, closing of orders, popping from queues
		 */
		askDone, bidDone = ob.makeTrade(ask, matchingBid, true)
		addBackOrders()

		// Check if error occurred in acquiring locks or database transactions
		if askDone == false && bidDone == false {
			l.Errorf("PerformOrderFillTransaction returned both askDone, bidDone false")
			return
		}

		// check if ask is fulfilled
		if askDone == true {
			break
		}

		matchingBid, addBackOrders = ob.getTopMatchingBid(ask)
	}

	// if ask is still not fulfilled, add it to queue
	if askDone == false {
		ob.addAskToQueue(ask)
	}
}

// processBid tries to match an incoming bid with existing asks
// and carry out a trade if possible
func (ob *orderBook) processBid(bid *models.Bid) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method":   "processBid",
		"paramBid": bid,
	})

	// in case of stoploss order, add it to queue and return
	if bid.OrderType == models.StopLoss {
		l.Debugf("Adding stopLoss with bid_id %d to the queue", bid.Id)
		ob.bidStoploss.Push(bid, bid.Price, bid.StockQuantity)
		return
	}

	// if control reaches here, it's NOT a stoploss order
	var askDone, bidDone bool
	matchingAsk, addBackOrders := ob.getTopMatchingAsk(bid)

	for matchingAsk != nil {
		/*
		 * makeTrade invokes fillOrderFn
		 * It handles changes in market depth, closing of orders, popping from queues
		 */
		askDone, bidDone = ob.makeTrade(matchingAsk, bid, false)
		addBackOrders()

		// Check if error occurred in acquiring locks or database transactions
		if askDone == false && bidDone == false {
			l.Errorf("PerformOrderFillTransaction returned both askDone, bidDone false")
			return
		}

		// check if bid is fulfilled
		if bidDone == true {
			break
		}
		matchingAsk, addBackOrders = ob.getTopMatchingAsk(bid)
	}

	// if bid is still not fulfilled, add it to queue
	if bidDone == false {
		ob.addBidToQueue(bid)
	}
}

// makeTrade performs a trade between an existing order and an incoming order
// isAsk is true for an incoming ask, and false for an incoming bid
func (ob *orderBook) makeTrade(ask *models.Ask, bid *models.Bid, isAsk bool) (bool, bool) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method":   "makeTrade",
		"paramAsk": ask,
		"paramBid": bid,
	})
	/*
	*	fillOrderFn should
	*		- acquire locks on the users
	*		- update StockQuantityFulfilled and IsClosed
	*		- record the transaction in the database
	 */
	stockTradePrice, stockTradeQty := getTradePriceAndQty(ask, bid)
	askDone, bidDone, tr := fillOrderFn(ask, bid, stockTradePrice, stockTradeQty)

	if tr != nil {
		l.Infof("Trade made between ask_id %d and bid %d at price %d", ask.Id, bid.Id, tr.Price)
		// tr is always AskTransaction. So its StockQty < 0. Make it positive.
		ob.depth.AddTrade(tr.Price, uint32(-tr.StockQuantity), tr.CreatedAt)

		// if ask is incoming, close just bid depth as ask hasn't even been added to depth
		if isAsk {
			ob.depth.CloseOrder(isMarket(bid.OrderType), false, bid.Price, uint32(-tr.StockQuantity))
		} else {
			ob.depth.CloseOrder(isMarket(ask.OrderType), true, ask.Price, uint32(-tr.StockQuantity))
		}

		// Trigger stop losses here
		ob.triggerStopLosses(tr)
	}

	if askDone {
		// If transaction didn't happen, but askDone is true
		// Thus the ask was faulty in some way or the order was cancelled
		// Thus, the remaining stock quantity needs to be closed
		if tr == nil {
			ob.depth.CloseOrder(isMarket(ask.OrderType), true, ask.Price, ask.StockQuantity-ask.StockQuantityFulfilled)
		}

		// if ask is incoming i.e isAsk = true, it hasn't even been added to queue
		if isAsk == false {
			ob.asks.Pop()
		}
	}
	if bidDone {
		// If transaction didn't happen, but bidDone is true
		// Thus the bid was faulty in some way or the order was cancelled
		// Thus, the remaining stock quantity needs to be closed
		if tr == nil {
			ob.depth.CloseOrder(isMarket(bid.OrderType), false, bid.Price, bid.StockQuantity-bid.StockQuantityFulfilled)
		}

		// if bid is incoming i.e isAsk = false, it hasn't even been added to queue
		if isAsk == true {
			ob.bids.Pop()
		}
	}

	return askDone, bidDone
}

/*
 *	Method to check and trigger(if possible) StopLoss orders whenever a transaction occurs
 */
func (ob *orderBook) triggerStopLosses(tr *models.Transaction) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method": "triggerStopLosses",
	})

	db := utils.GetDB()

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
		topBidStoploss = ob.bidStoploss.Head()
	}
}

/*
 *	Method to clear the existing orders for a particular stock
 */
func (ob *orderBook) clearExistingOrders() {
	var l = ob.logger.WithFields(logrus.Fields{
		"method": "clearExistingOrders",
	})

	if ob.bids.Head() == nil {
		return
	}

	var askDone, bidDone bool

	bidTop := ob.bids.Pop()
	askTop, addBackOrders := ob.getTopMatchingAsk(bidTop)

	for bidTop != nil && askTop != nil {
		// passing false as third argument
		// thus, treating bid as incoming order
		// hence bid should NOT be in queue, which is why the bid has been popped
		askDone, bidDone = ob.makeTrade(askTop, bidTop, false)
		addBackOrders()

		// Check if error occurred in acquiring locks or database transactions
		if askDone == false && bidDone == false {
			l.Errorf("PerformOrderFillTransaction returned both askDone, bidDone false")
			return
		}

		// if current bid is fulfilled, move to next bid
		if bidDone == true {
			bidTop = ob.bids.Pop()
		}
		if bidTop != nil {
			askTop, addBackOrders = ob.getTopMatchingAsk(bidTop)
		}
	}

	// if bid is not fulfilled and asks queue becomes empty
	if bidTop != nil {
		ob.addBidToQueue(bidTop)
	}
}

/*
 *	Method to wait for an incoming order on the channels
 */
func (ob *orderBook) waitForOrder() {
	var l = ob.logger.WithFields(logrus.Fields{
		"method": "waitForOrder",
	})

	select {
	case askOrder := <-ob.askChan:
		l.Debugf("Got ask %+v. Processing", askOrder)
		ob.processAsk(askOrder)

	case bidOrder := <-ob.bidChan:
		l.Debugf("Got bid %+v. Processing", bidOrder)
		ob.processBid(bidOrder)
	}
}
