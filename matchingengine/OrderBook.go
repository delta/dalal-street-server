package matchingengine

import (
	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// FillOrder is a type definition for a function that fills an order with given ask, bid, stockPrice and stockQty
type FillOrder func(ask *models.Ask, bid *models.Bid, stockTradePrice uint32, stockTradeQty uint32) (models.AskOrderFillStatus, models.BidOrderFillStatus, *models.Transaction)

// fillOrderFn is the actual function that filles an order.
// It has been separated from implementation to ease testing.
var fillOrderFn FillOrder = models.PerformOrderFillTransaction

// OrderBook stores the order book for a given stock
type OrderBook interface {
	LoadOldTransactions(txs []*models.Transaction)
	LoadOldAsk(*models.Ask)
	LoadOldBid(*models.Bid)
	AddAskOrder(*models.Ask)
	AddBidOrder(*models.Bid)
	CancelAskOrder(*models.Ask)
	CancelBidOrder(*models.Bid)
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

// addAskToDepth adds an ask to the depth
// NOTE: 1. Only adds unfulfilled qty
//		 2. It uses ask.StockQuantityFulfilled which gets written to by fillOrderFn, but this
//			function doesn't run concurrently with fillOrderFn
func (ob *orderBook) addAskToDepth(ask *models.Ask) {
	// use unfulfilled qty here, because depth should have only the unfulfilled qty
	askUnfulfilledQuantity := ask.StockQuantity - ask.StockQuantityFulfilled
	ob.depth.AddOrder(isMarket(ask.OrderType), true, ask.Price, askUnfulfilledQuantity)
}

// addBidToDepth adds a bid to the depth.
// NOTE: 1. Only adds unfulfilled qty
//		 2. It uses ask.StockQuantityFulfilled which gets written to by fillOrderFn, but this
//			function doesn't run concurrently with fillOrderFn
func (ob *orderBook) addBidToDepth(bid *models.Bid) {
	// use unfulfilled qty here, because depth should have only the unfulfilled qty
	bidUnfulfilledQuantity := bid.StockQuantity - bid.StockQuantityFulfilled
	ob.depth.AddOrder(isMarket(bid.OrderType), false, bid.Price, bidUnfulfilledQuantity)
}

func (ob *orderBook) LoadOldTransactions(txs []*models.Transaction) {
	for _, tx := range txs {
		ob.depth.AddTrade(tx.Price, uint32(-tx.StockQuantity), tx.CreatedAt)
	}
}

// LoadOldAsk loads an old ask into the order book
// NOTE: 1. If it's a stoploss order it adds to the stoploss queue
// 		 2. Otherwise it adds it to the regular queue, and updates depth
func (ob *orderBook) LoadOldAsk(ask *models.Ask) {
	l := ob.logger.WithFields(logrus.Fields{
		"method":      "LoadOldAsk",
		"param_askId": ask.Id,
	})

	// in case of stoploss order, add it to stoploss queue and return
	if ask.OrderType == models.StopLoss {
		l.Debugf("Adding stopLoss with ask_id %d to the queue", ask.Id)
		ob.askStoploss.Push(ask)
		return
	}

	ob.asks.Push(ask)
	ob.addAskToDepth(ask)
}

// LoadOldBid loads an old bid into the order book
// NOTE: 1. If it's a stoploss order it adds to the stoploss queue
// 		 2. Otherwise it adds it to the regular queue, and updates depth
func (ob *orderBook) LoadOldBid(bid *models.Bid) {
	l := ob.logger.WithFields(logrus.Fields{
		"method":      "LoadOldBid",
		"param_bidId": bid.Id,
	})

	// in case of stoploss order, add it to stoploss queue and return
	if bid.OrderType == models.StopLoss {
		l.Debugf("Adding stopLoss with bid_id %d to the queue", bid.Id)
		ob.bidStoploss.Push(bid)
		return
	}

	ob.bids.Push(bid)
	ob.addBidToDepth(bid)
}

// AddAskOrder adds a NEW ask order to the book. It will take care of adding stopLoss.
// It should NOT be passed partially filled orders
func (ob *orderBook) AddAskOrder(ask *models.Ask) {
	l := ob.logger.WithFields(logrus.Fields{
		"method":      "AddAskOrder",
		"param_bidId": ask.Id,
	})

	// in case of stoploss order, add it to stoploss queue and return
	if ask.OrderType == models.StopLoss {
		l.Debugf("Adding stopLoss with ask_id %d to the queue", ask.Id)
		ob.askStoploss.Push(ask)
		return
	}

	// otherwise synchronize calls to processAsk(ask)
	ob.askChan <- ask
}

// AddBidOrder adds a NEW bid order to the book. It will take care of adding stopLoss.
// It should NOT be passed partially fulfilled orders
func (ob *orderBook) AddBidOrder(bid *models.Bid) {
	l := ob.logger.WithFields(logrus.Fields{
		"method":      "AddBidOrder",
		"param_bidId": bid.Id,
	})

	// in case of stoploss order, add it to queue and return
	if bid.OrderType == models.StopLoss {
		l.Debugf("Adding stopLoss with bid_id %d to the queue", bid.Id)
		ob.bidStoploss.Push(bid)
		return
	}

	// otherwise synchronize calls to processBid(bid)
	ob.bidChan <- bid
}

// CancelAskOrder removes an Ask Order from the OrderBook
// NOTE: 1. It *ONLY* removes it from the depth, and not from the queue
// 			Removal from queue will happen when the order gets selected for trading
// 			at some point in time. This is inefficient and needs to be fixed by moving
// 			from pqueue to a better data structure to handle orders.
//		 2. It accesses StockQuantityFulfilled but this will be called only after
//		    models.CancelOrder has been called, so the ask's properties won't be modified now
func (ob *orderBook) CancelAskOrder(ask *models.Ask) {
	unfulfilled := ask.StockQuantity - ask.StockQuantityFulfilled
	ob.depth.CloseOrder(isMarket(ask.OrderType), true, ask.Price, unfulfilled)
}

// CancelBidOrder removes an Bid Order from the OrderBook
// NOTE: 1. It *ONLY* removes it from the depth, and not from the queue
// 			Removal from queue will happen when the order gets selected for trading
// 			at some point in time. This is inefficient and needs to be fixed by moving
// 			from pqueue to a better data structure to handle orders
// 		 2. It accesses StockQuantityFulfilled but this will be called only after
//		    models.CancelOrder has been called, so the bid's properties won't be modified now
func (ob *orderBook) CancelBidOrder(bid *models.Bid) {
	unfulfilled := bid.StockQuantity - bid.StockQuantityFulfilled
	ob.depth.CloseOrder(isMarket(bid.OrderType), false, bid.Price, unfulfilled)
}

/**
 *	StartStockMatching listens for incoming orders for a particular stock and process them.
 *  NOTE: It will spawn a new go routine. It needn't be run like "go ob.StartStockMatching()".
 *        This function will return when it has loaded old orders and cleared existing those.
 */
func (ob *orderBook) StartStockMatching() {
	var l = ob.logger.WithFields(logrus.Fields{
		"method": "startStockMatching",
	})

	l.Infof("Started with ask_count = %d, bid_count = %d", ob.asks.Size(), ob.bids.Size())

	// Clear the existing orders first
	ob.clearExistingOrders()

	// run infinite loop
	go func() {
		for {
			ob.waitForOrder()
		}
	}()
}

// getTopMatchingBid checks for a matching bid from the orderbook for an incoming ask
// NOTE: 1. It does NOT remove the bid from the queue.
//       2. It does NOT update the market depth
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
			ob.bids.Push(bid)
			// do not add to the market depth!
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

// getTopMatchingAsk returns a matching ask from the orderbook for an incoming bid
// NOTE: 1. It does NOT remove the ask from the queue
//       2. It does NOT update the market depth
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
			ob.asks.Push(ask)
			// do not add to the market depth!
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
// and carry out a trade if possible.
// NOTE: 0. The ask shouldn't be a stoploss. It's either market, or limit.
// 		 1. The ask hasn't been added to the queue or to the depth
// 	     2. After dealing with all possible matching bids, if the ask
//          is still not fulfilled, it is put in the queue, and depth gets updated.
//		 3. For every matching bid that is handled for this ask,
//			the bid will be removed if it's either closed already, or if it got
//			fulfilled, or if the buyer doesn't have enough cash to fulfill the
//			trade.
func (ob *orderBook) processAsk(ask *models.Ask) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method":   "processAsk",
		"paramAsk": ask,
	})

	var askDone, bidDone bool

	// matchingBid is still in the queue. It must be removed once it finishes
	matchingBid, addBackOrders := ob.getTopMatchingBid(ask)

	for matchingBid != nil {
		/*
		 * makeTrade invokes fillOrderFn
		 * It handles changes in market depth, closing of orders
		 * It does not handle popping of the opposing order
		 */
		askDone, bidDone = ob.makeTrade(ask, matchingBid, true, false)
		if bidDone {
			ob.bids.Pop()
		}
		addBackOrders()

		// Check if error occurred in acquiring locks or database transactions
		if askDone == false && bidDone == false {
			l.Errorf("makeTrade returned both askDone, bidDone false")
			return
		}

		// check if ask is fulfilled
		if askDone == true {
			// nothing more needs to be done here. addBackOrders has been called
			// and if the bid was fulfilled, it has been popped above already.
			return
		}

		matchingBid, addBackOrders = ob.getTopMatchingBid(ask)
	}

	// addBackOrders doesn't need to be called here because matchingBid is nil

	// if ask is still not fulfilled, add it to queue & update depth
	if askDone == false {
		ob.asks.Push(ask)
		ob.addAskToDepth(ask)
	}
}

// processBid tries to match an incoming bid with existing asks
// and carry out a trade if possible.
// NOTE: 0. The bid shouldn't be a stoploss. It's either market, or limit.
// 		 1. The bid hasn't been added to the queue or to the depth
// 	     2. After dealing with all possible matching asks, if the bid
//          is still not fulfilled, it is put in the queue and depth gets updated.
//		 3. For every matching ask that is handled for this bid,
//			the ask will be removed if it's either closed already, or if it got
//			fulfilled, or if the seller doesn't have enough stocks to fulfill the
//			trade.
func (ob *orderBook) processBid(bid *models.Bid) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method":   "processBid",
		"paramBid": bid,
	})

	// if control reaches here, it's NOT a stoploss order
	var askDone, bidDone bool

	// matchingAsk is still in the queue. It must be removed once it finishes
	matchingAsk, addBackOrders := ob.getTopMatchingAsk(bid)

	for matchingAsk != nil {
		/*
		 * makeTrade invokes fillOrderFn
		 * It handles changes in market depth, closing of orders
		 * It does not handle popping of the opposing order
		 */
		askDone, bidDone = ob.makeTrade(matchingAsk, bid, false, true)
		if askDone {
			ob.asks.Pop()
		}
		addBackOrders()

		// Check if error occurred in acquiring locks or database transactions
		if askDone == false && bidDone == false {
			l.Errorf("makeTrade returned both askDone, bidDone false")
			return
		}

		// check if bid is fulfilled
		if bidDone == true {
			// nothing more needs to be done here. addBackOrders has been called
			// and if the ask was fulfilled, it has been popped above already.
			return
		}
		matchingAsk, addBackOrders = ob.getTopMatchingAsk(bid)
	}

	// addBackOrders doesn't need to be called here because matchingAsk is nil

	// if bid is still not fulfilled, add it to queue & update depth
	if bidDone == false {
		ob.bids.Push(bid)
		ob.addBidToDepth(bid)
	}
}

// makeTrade performs a trade between two orders
// NOTE: 0. incoming-ness is only used to decide whether to update depth. Non-incomers update depth.
//		 1. incomingAsk/Bid is true if the ask/bid is incoming - ie not added to queue yet
//       2. Both incomingAsk and incomingBid SHOULDN'T be true. Usually exactly one
//			is true, but when clearing existing orders, both will be false
//       3. It returns two booleans: askDone, bidDone. Each is true if the ask/bid order has been
//     		closed and is safe to remove it from the queue.
//		 4. The *CALLER* is responsible for dealing with the queue (adding/removing)
//			[except for triggerStoploss]
//		 5. If transaction happens:
//			a). A new trade is added to the depth.
//		 	b). Stoplossess get triggered
//			c). Non-incoming order(s) are removed from depth where removed qty==tr.qty
//		 6. If transaction doesn't happen:
//			a). Non-incoming order(s) are removed from depth where removed qty==unfulfilled.
//				This is done ONLY if the order wasn't already closed (See below)
//
// NOTE: Whether the order was closed already has to be decided by the fillOrderFn
// as accessing the IsClosed attribute of the Bid from here is unsafe. It
// could lead to a race condition between OrderBook and Users.CancelOrder()
//
// ----------------------------------------------------------------
// The current approach to dealing with cancellations is this:
//
// fillOrderFn and models.CancelOrder will *not* run concurrently if a common user is involved.
// Therefore, fillOrderFn and models.CancelOrder are *strongly ordered*. One happens before
// the other.
//
// So, if fillOrderFn returns orderstatus as AlreadyClosed, then that means
// models.CancelOrder executed before it for that order. Since models.CancelOrder will be followed
// by OrderBook.CancelOrder, makeTrade doesn't handle that case. Instead, OrderBook.CancelOrder will
// be called sometime (soon, or even concurrently with makeTrade), and it'll remove the required amount
// of stockqty from depth.

// Basically, makeTrade will update depth only when it's a non-incoming order
// AND the orderstatus is either Done or Undone, but not when it's AlreadyClosed
func (ob *orderBook) makeTrade(ask *models.Ask, bid *models.Bid, incomingAsk bool, incomingBid bool) (bool, bool) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method":   "makeTrade",
		"paramAsk": ask,
		"paramBid": bid,
	})

	/*
	 *	fillOrderFn should
	 *      - acquire locks on the users
	 *		- update StockQuantityFulfilled and IsClosed
	 *		- record the transaction in the database
	 */

	stockTradePrice, stockTradeQty := getTradePriceAndQty(ask, bid)
	askStatus, bidStatus, tr := fillOrderFn(ask, bid, stockTradePrice, stockTradeQty)

	if tr != nil {
		l.Infof("Trade made between ask_id %d and bid %d at price %d", ask.Id, bid.Id, tr.Price)
		// tr is always AskTransaction. So its StockQty < 0. Make it positive.
		ob.depth.AddTrade(tr.Price, uint32(-tr.StockQuantity), tr.CreatedAt)

		// if ask is incoming, close just bid depth as ask hasn't even been added to depth
		if !incomingBid {
			ob.depth.CloseOrder(isMarket(bid.OrderType), false, bid.Price, uint32(-tr.StockQuantity))
		}
		if !incomingAsk {
			ob.depth.CloseOrder(isMarket(ask.OrderType), true, ask.Price, uint32(-tr.StockQuantity))
		}

		// Trigger stop losses here
		ob.triggerStopLosses(tr)
	} else {
		// If transaction didn't happen, but bidDone is true
		// Thus the bid was faulty in some way or the order was cancelled
		// Thus, the remaining stock quantity needs to be closed
		// If the bid was already closed, then CancelOrder was called sometime before fillOrderFn
		// was called. That means orderBook.CancelBidOrder would (have been)/(be) called. It will
		// handle updating the depth.
		if !incomingBid && bidStatus != models.BidAlreadyClosed {
			unfulfilled := bid.StockQuantity - bid.StockQuantityFulfilled
			ob.depth.CloseOrder(isMarket(bid.OrderType), false, bid.Price, unfulfilled)
		}
		// If transaction didn't happen, but askDone is true
		// Thus the ask was faulty in some way or the order was cancelled
		// Thus, the remaining stock quantity needs to be closed
		// If the ask was already closed, then CancelOrder was called sometime before fillOrderFn
		// was called. That means orderBook.CancelAskOrder would (have been)/(be) called. It will
		// handle updating the depth.
		if !incomingAsk && askStatus != models.AskAlreadyClosed {
			unfulfilled := ask.StockQuantity - ask.StockQuantityFulfilled
			ob.depth.CloseOrder(isMarket(ask.OrderType), true, ask.Price, unfulfilled)
		}
	}

	return askStatus != models.AskUndone, bidStatus != models.BidUndone
}

/*
 *	Method to check and trigger(if possible) StopLoss orders whenever a transaction occurs
 */
func (ob *orderBook) triggerStopLosses(tr *models.Transaction) {
	var l = ob.logger.WithFields(logrus.Fields{
		"method": "triggerStopLosses",
	})

	l.Debugf("Triggering ask stoplosses")
	topAskStoploss := ob.askStoploss.Head()
	// We trigger asks having higher than current price. That's because if it's an ask stoploss, then we will sell
	// if the current price goes below the trigger price.
	for topAskStoploss != nil && tr.Price <= topAskStoploss.Price {
		l.Debugf("Triggering ask %+v", topAskStoploss)

		topAskStoploss = ob.askStoploss.Pop()
		if err := topAskStoploss.TriggerStoploss(); err != nil {
			l.Errorf("Error while changing state of %+v from StopLoss to StopLossActive", topAskStoploss)
		}

		ob.asks.Push(topAskStoploss)
		// do not add to the market depth. It's stoplossactive, meaning market order now
		topAskStoploss = ob.askStoploss.Head()
	}

	l.Debugf("Triggering bid stoplosses")
	// We trigger bids having lower than current price. That's because if it's a bid stoploss, we will buy if the
	// current price goes above the trigger price.
	topBidStoploss := ob.bidStoploss.Head()
	for topBidStoploss != nil && tr.Price >= topBidStoploss.Price {
		l.Debugf("Triggering bid %+v", topBidStoploss)

		topBidStoploss = ob.bidStoploss.Pop()
		if err := topBidStoploss.TriggerStoploss(); err != nil {
			l.Errorf("Error while changing state of %+v from StopLoss to StopLossActive", topBidStoploss)
		}

		ob.bids.Push(topBidStoploss)
		// do not add to the market depth. It's stoplossactive, meaning market order now
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
		// treating both ask and bid as non-incoming orders
		// which means depth will be updated for both.
		askDone, bidDone = ob.makeTrade(askTop, bidTop, false, false)
		addBackOrders()

		// Check if error occurred in acquiring locks or database transactions
		if askDone == false && bidDone == false {
			l.Errorf("makeTrade returned both askDone, bidDone false")
			return
		}

		// if ask is done, remove it. Depth is already updated. No need to worry.
		if askDone {
			ob.asks.Pop()
		}
		// if current bid is fulfilled, move to next bid
		if bidDone {
			bidTop = ob.bids.Pop()
		}
		if bidTop != nil {
			// this will work even when askDone = false, bidDone = true including
			// the weird case where same user bids-asks get involved
			askTop, addBackOrders = ob.getTopMatchingAsk(bidTop)
		}
		// otherwise the loop will break. Since askTop hasn't been popped from
		// asks queue, so no need to worry about it
	}

	// if bid is not fulfilled and asks queue becomes empty. Shouldn't update depth
	if bidTop != nil {
		ob.bids.Push(bidTop)
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
