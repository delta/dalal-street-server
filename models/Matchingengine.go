package models

import (
	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/datastreams"
)

type OrderBook struct {
	stockId     uint32
	askChan     chan *Ask
	bidChan     chan *Bid
	asks        *AskPQueue
	bids        *BidPQueue
	askStoploss *AskPQueue
	bidStoploss *BidPQueue
	depth       *datastreams.MarketDepth
}

/*
 *	Function to return the minimum of two numbers
 */
func min(i, j uint32) uint32 {
	if i < j {
		return i
	}
	return j
}

/*
 *	Helper function to check if it's a Market order
 */
func isMarket(oType OrderType) bool {
	return oType == Market || oType == StopLossActive
}

/*
 *	Helper function to check if a transaction is possible
 */
func isOrderMatching(askTop *Ask, bidTop *Bid) bool {
	if isMarket(bidTop.OrderType) || isMarket(askTop.OrderType) {
		return true
	}
	return bidTop.Price >= askTop.Price
}

/*
 *	Method to check and trigger(if possible) StopLoss orders whenever a transaction occurs
 */
func (ob *OrderBook) triggerStopLosses(tr *Transaction) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "triggerStopLosses",
		"stockId": ob.stockId,
	})

	db, err := DbOpen()
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
		topAskStoploss.OrderType = StopLossActive
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
		topBidStoploss.OrderType = StopLossActive
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
func (ob *OrderBook) getTopMatchingOrders() (*Ask, *Bid, func()) {
	askTop, bidTop := ob.asks.Head(), ob.bids.Head()

	// If either one is nil, there are no matching orders
	if askTop == nil || bidTop == nil {
		return askTop, bidTop, func() {}
	}

	// Store the same user's bids in a slice
	var sameUserBids []*Bid
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
func (ob *OrderBook) processOrder() {
	var l = logger.WithFields(logrus.Fields{
		"method":  "processOrder",
		"stockId": ob.stockId,
	})

	var (
		askDone bool
		bidDone bool
		tr      *Transaction
	)

	askTop, bidTop, addBackOrders := ob.getTopMatchingOrders()

	for bidTop != nil && askTop != nil {

		l.Debugf("Performing OrderFill transaction")

		/*
		 *	PerformOrderFillTransaction should
		 *		- acquire locks on the users
		 *		- update StockQuantityFulfilled and IsClosed
		 *		- record the transaction in the database
		 */
		askDone, bidDone, tr = PerformOrderFillTransaction(askTop, bidTop)

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
func (ob *OrderBook) waitForOrder() {
	var l = logger.WithFields(logrus.Fields{
		"method": "waitForOrder",
	})

	select {
	case askOrder := <-ob.askChan:
		l.Debugf("Got ask %+v. Processing", askOrder)
		if askOrder.OrderType == StopLoss {
			l.Debugf("Adding stopLoss with ask_id %d to the list", askOrder.Id)
			ob.askStoploss.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
			break
		}

		// If control reaches here, it's not a StopLoss order
		ob.asks.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
		ob.depth.AddOrder(isMarket(askOrder.OrderType), true, askOrder.Price, askOrder.StockQuantity)
		ob.processOrder()

	case bidOrder := <-ob.bidChan:
		l.Debugf("Got bid %+v. Processing", bidOrder)
		if bidOrder.OrderType == StopLoss {
			l.Debugf("Adding stopLoss with bid_id %d to the list", bidOrder.Id)
			ob.bidStoploss.Push(bidOrder, bidOrder.Price, bidOrder.StockQuantity)
			break
		}

		// If control reaches here, it's not a StopLoss order
		ob.bids.Push(bidOrder, bidOrder.Price, bidOrder.StockQuantity)
		ob.depth.AddOrder(isMarket(bidOrder.OrderType), true, bidOrder.Price, bidOrder.StockQuantity)
		ob.processOrder()
	}
}

/*
 *	Method to listen for incoming orders for a particular stock and process them
 */
func (ob *OrderBook) startStockMatching() {
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
	}
}

/*
 *	Store details of the placed orders. Each entry in orderBooks corresponds to a particular stock
 */
var orderBooks = make(map[uint32]OrderBook)

/*
 *	method to add the placed ask order to the ask channel. Called from method PlaceAskOrder
 */
func AddAskOrder(askOrder *Ask) {
	orderBooks[askOrder.StockId].askChan <- askOrder
}

/*
 *	method to add the placed bid order to the bid channel. Called from method PlaceBidOrder
 */
func AddBidOrder(bidOrder *Bid) {
	orderBooks[bidOrder.StockId].bidChan <- bidOrder
}

/*
 *	Init will be run once when server is started
 *	It calls StartStockmatching for all the stocks in concurrent goroutines
 */
func InitMatchingEngine() {
	var l = logger.WithFields(logrus.Fields{
		"method": "InitMatchingEngine",
	})
	db, err := DbOpen()
	if err != nil {
		l.Errorf("Errored : %+v", err)
		panic("Error opening database for matching engine")
	}
	defer db.Close()

	var (
		openAskOrders          []*Ask
		openBidOrders          []*Bid
		stockIds               []uint32
		askUnfulfilledQuantity uint32
		bidUnfulfilledQuantity uint32
	)

	//Load stock ids from database
	if err := db.Model(&Stock{}).Pluck("id", &stockIds).Error; err != nil {
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
		orderBooks[stockId] = OrderBook{
			stockId:     stockId,
			askChan:     make(chan *Ask),
			bidChan:     make(chan *Bid),
			asks:        NewAskPQueue(MINPQ), //lower price has higher priority
			bids:        NewBidPQueue(MAXPQ), //higher price has higher priority
			askStoploss: NewAskPQueue(MAXPQ), // stoplosses work like opposite of limit/market.
			bidStoploss: NewBidPQueue(MINPQ), // They sell when price goes below a certain trigger price.
			depth:       datastreams.NewMarketDepth(stockId),
		}
	}

	//Load open ask orders into priority queue
	for _, openAskOrder := range openAskOrders {
		if openAskOrder.OrderType == StopLoss {
			orderBooks[openAskOrder.StockId].askStoploss.Push(openAskOrder, openAskOrder.Price, 0)
		} else {
			askUnfulfilledQuantity = openAskOrder.StockQuantity - openAskOrder.StockQuantityFulfilled
			orderBooks[openAskOrder.StockId].asks.Push(openAskOrder, openAskOrder.Price, askUnfulfilledQuantity)
		}
	}

	//Load open bid orders into priority queue
	for _, openBidOrder := range openBidOrders {
		if openBidOrder.OrderType == StopLoss {
			orderBooks[openBidOrder.StockId].bids.Push(openBidOrder, openBidOrder.Price, 0)
		} else {
			bidUnfulfilledQuantity = openBidOrder.StockQuantity - openBidOrder.StockQuantityFulfilled
			orderBooks[openBidOrder.StockId].bids.Push(openBidOrder, openBidOrder.Price, bidUnfulfilledQuantity)
		}
	}

	for _, ob := range orderBooks {
		go ob.startStockMatching()
	}
}
