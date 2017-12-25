package models

import (
	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
)

type StockDetails struct {
	askChan     chan *Ask
	bidChan     chan *Bid
	asks        *AskPQueue
	bids        *BidPQueue
	askStoploss *AskPQueue
	bidStoploss *BidPQueue
	depth       *datastreams.MarketDepth
}

/*
*	map to store details of the placed orders. Maps stockId to the PlacedOrderDetails for that stockId
 */
var stocks = make(map[uint32]StockDetails)

/*
*	method to add the placed ask order to the ask channel. Called from method PlaceAskOrder
 */
func AddAskOrder(askOrder *Ask) {
	stocks[askOrder.StockId].askChan <- askOrder
}

/*
*	method to add the placed bid order to the bid channel. Called from method PlaceBidOrder
 */
func AddBidOrder(bidOrder *Bid) {
	stocks[bidOrder.StockId].bidChan <- bidOrder
}

/*
*	primary function to perform matching and transaction
 */
func StartStockMatching(stock StockDetails, stockId uint32) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "StartStockMatching",
		"stockId": stockId,
	})

	l.Infof("Started %d %d", stock.asks.Size(), stock.bids.Size())

	// Won't be called concurrently.
	var triggerStoplosses = func(tr *Transaction) {
		db, err := DbOpen()
		if err != nil {
			l.Errorf("Error while opening DB to trigger stoplosses. Not triggering them.")
			return
		}
		defer db.Close()

		l.Debugf("Triggering ask stoplosses")
		topAskStoploss := stocks[stockId].askStoploss.Head()
		// We trigger asks having higher than current price. That's because if it's an ask stoploss, then we will sell
		// if the current price goes below the trigger price.
		for topAskStoploss != nil && tr.Price <= topAskStoploss.Price {
			l.Debugf("Triggering ask %+v", topAskStoploss)

			topAskStoploss = stocks[stockId].askStoploss.Pop()
			topAskStoploss.OrderType = StopLossActive
			if err := db.Save(topAskStoploss).Error; err != nil {
				l.Errorf("Error while changing state of %+v from StopLoss to StopLossActive", topAskStoploss)
			}

			stocks[stockId].asks.Push(topAskStoploss, topAskStoploss.Price, topAskStoploss.StockQuantity)
			stocks[stockId].depth.AddOrder(true, true, topAskStoploss.Price, topAskStoploss.StockQuantity)
			topAskStoploss = stocks[stockId].askStoploss.Head()
		}

		l.Debugf("Triggering bid stoplosses")
		// We trigger bids having lower than current price. That's because if it's a bid stoploss, we will buy if the
		// current price goes above the trigger price.
		topBidStoploss := stocks[stockId].bidStoploss.Head()
		for topBidStoploss != nil && tr.Price >= topBidStoploss.Price {
			l.Debugf("Triggering bid %+v", topBidStoploss)

			topBidStoploss = stocks[stockId].bidStoploss.Pop()
			topBidStoploss.OrderType = StopLossActive
			if err := db.Save(topBidStoploss).Error; err != nil {
				l.Errorf("Error while changing state of %+v from StopLoss to StopLossActive", topBidStoploss)
			}

			stocks[stockId].bids.Push(topBidStoploss, topBidStoploss.Price, topBidStoploss.StockQuantity)
			stocks[stockId].depth.AddOrder(true, false, topBidStoploss.Price, topBidStoploss.StockQuantity)
			topBidStoploss = stocks[stockId].bidStoploss.Head()
		}
	}

	/*
	*	processAsk is guaranteed exclusive access to stocks[]
	*
	*	processAsk performs the following functions
	*		- get the top bid order. If there are no bids for the current stock, store the ask and return
	*		- acquire lock on askingUser and biddingUser in order of user ids
	*		- check if price of ask is less than that of the top bid. If no, and the ask is still not satisfied,
	*		  store the ask and return
	*		- call PerformOrderFillTransaction to carry out the transaction
	*
	*	Possible return values
	*		- if donefornow is true, no more iterations are required. Either the ask has been completely satisfied or there
	*		  are no matching bids for now or PerformOrderFillTransaction has failed
	*		- if donefornow is false, continue iterations. Either some error has occured, or the topBid has been popped
	 */
	var processAsk = func(askOrder *Ask) (donefornow bool) {
		var l = logger.WithFields(logrus.Fields{
			"method":         "processAsk",
			"param_askOrder": askOrder,
		})

		l.Infof("Attempting")

		topBidOrder := stocks[askOrder.StockId].bids.Head()
		depth := stocks[askOrder.StockId].depth

		l.Debugf("Ignoring same user's asks")

		// Don't sell a user's stocks to himself.
		var sameUserBids []*Bid
		for topBidOrder != nil && topBidOrder.UserId == askOrder.UserId {
			sameUserBids = append(sameUserBids, topBidOrder)
			topBidOrder = stocks[askOrder.StockId].bids.Pop()
		}
		defer func() {
			for _, sub := range sameUserBids {
				stocks[askOrder.StockId].bids.Push(sub, sub.Price, sub.StockQuantity)
			}
		}()

		if topBidOrder == nil {
			l.Debugf("No matching top bid order currently. Adding to orderbook")
			stocks[askOrder.StockId].asks.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
			depth.AddOrder(askOrder.OrderType == Market, true, askOrder.Price, askOrder.StockQuantity)
			return true
		}

		l.Debugf("TopBidOrder found as: %+v", topBidOrder)
		l.Debugf("Acquiring lock in order of User Ids")

		var firstUserId uint32
		var secondUserId uint32
		var isAskFirst bool

		//look out for error!!!
		if askOrder.UserId < topBidOrder.UserId {
			firstUserId = askOrder.UserId
			secondUserId = topBidOrder.UserId
			isAskFirst = true
		} else {
			firstUserId = topBidOrder.UserId
			secondUserId = askOrder.UserId
			isAskFirst = false
		}

		l.Debugf("Want first and second as %d, %d for stockid: %d", firstUserId, secondUserId, stockId)
		defer l.Debugf("Closed channels of %d and %d for stockid: %d", firstUserId, secondUserId, stockId)

		firstLockChan, firstUser, err := getUser(firstUserId)
		if err != nil {
			l.Errorf("Errored: %+v", err)
			return false
		}
		defer close(firstLockChan)

		secondLockChan, secondUser, err := getUser(secondUserId)
		if err != nil {
			l.Errorf("Errored: %+v", err)
			return false
		}
		defer close(secondLockChan)

		l.Debugf("Acquired the locks on users %d and %d", firstUserId, secondUserId)

		if !isAskOrderMatching(askOrder) {
			stockQuantityYetToBeFulfilled := askOrder.StockQuantity - askOrder.StockQuantityFulfilled
			l.Debugf("Unable to find the match. Putting ask in orderbook. Unfulfilled qty: %d", stockQuantityYetToBeFulfilled)
			if !askOrder.IsClosed {
				stocks[askOrder.StockId].asks.Push(askOrder, askOrder.Price, stockQuantityYetToBeFulfilled)
				depth.AddOrder(askOrder.OrderType == Market, true, askOrder.Price, stockQuantityYetToBeFulfilled)
			}
			return true
		}

		l.Debugf("Perform OrderFill Transaction")

		var (
			askDone bool
			bidDone bool
			tr      *Transaction
		)

		l.Debugf("Performing OrderFill transaction")
		//PerformOrderFillTransaction should update StockQuantityFulfilled and IsClosed
		if isAskFirst {
			askDone, bidDone, tr = PerformOrderFillTransaction(firstUser, secondUser, askOrder, topBidOrder)
		} else {
			askDone, bidDone, tr = PerformOrderFillTransaction(secondUser, firstUser, askOrder, topBidOrder)
		}

		/*
			If a transaction was made, remove that much quantity from bid
			also register a trade.
			Otherwise if the bid is done, remove whatever is unfulfilled from the depth
		*/
		if tr != nil {
			l.Infof("Trade made between ask and bid (%+v)", topBidOrder)
			// tr is always AskTransaction. So its StockQty < 0. Make it positive.
			depth.Trade(tr.Price, uint32(-tr.StockQuantity), tr.CreatedAt)
			//depth.CloseOrder(true, askOrder.Price, tr.StockQuantity) - don't! Haven't added ask to depth
			depth.CloseOrder(topBidOrder.OrderType == Market, false, topBidOrder.Price, uint32(-tr.StockQuantity))

			// Trigger all stoplosses that can be triggered
			triggerStoplosses(tr)
		} else {
			l.Infof("Trade not made. AskDone = %+v, BidDone = %v", askDone, bidDone)
			/*
				if askDone {
					 do nothing. We haven't even added the ask to the depth
				}
			*/
			if bidDone {
				depth.CloseOrder(topBidOrder.OrderType == Market, false, topBidOrder.Price, topBidOrder.StockQuantity-topBidOrder.StockQuantityFulfilled)
			}
		}

		// If ask not done, there's nothing to do.
		//if !askDone {
		//stockQuantityYetToBeFulfilled := askOrder.StockQuantity - askOrder.StockQuantityFulfilled
		//stocks[askOrder.StockId].asks.Push(askOrder, askOrder.Price, stockQuantityYetToBeFulfilled)
		//}

		// if askDone {
		// 	return true, nil
		// } else if bidDone {
		// 	stocks[askOrder.StockId].bids.Pop()
		// 	return false, nil
		// }
		if bidDone {
			l.Debugf("Popping topBidOrder %+v", topBidOrder)
			stocks[askOrder.StockId].bids.Pop()
		}
		if askDone {
			return true
		} else {
			return false
		}

		l.Errorf("PerformOrderFillTransaction returned both askDone, bidDone false")
		return true
	}

	/*
	*	processBid is guaranteed exclusive access to stocks[]
	*
	*	processBid performs the following functions
	*		- get the top ask order. If there are no asks for the current stock, store the bid and return
	*		- acquire lock on askingUser and biddingUser in order of user ids
	*		- check if price of bid is greater than that of the top ask. If no, and the bid is still not satisfied,
	*		  store the bid and return
	*		- call PerformOrderFillTransaction to carry out the transaction
	*
	*	Possible return values
	*		- if donefornow is true, no more iterations are required. Either the bid has been completely satisfied or there
	*		  are no matching asks for now or PerformOrderFillTransaction has failed
	*		- if donefornow is false, continue iterations. Either some error has occured, or the topAsk has been popped
	 */
	var processBid = func(bidOrder *Bid) (donefornow bool) {
		var l = logger.WithFields(logrus.Fields{
			"method":         "processBid",
			"param_bidOrder": bidOrder,
		})

		l.Infof("Attempting")

		topAskOrder := stocks[bidOrder.StockId].asks.Head()
		depth := stocks[bidOrder.StockId].depth

		l.Debugf("Ignoring same user's bids")

		// Don't sell a user's stocks to himself.
		var sameUserAsks []*Ask
		for topAskOrder != nil && topAskOrder.UserId == bidOrder.UserId {
			sameUserAsks = append(sameUserAsks, topAskOrder)
			topAskOrder = stocks[bidOrder.StockId].asks.Pop()
		}
		defer func() {
			for _, sua := range sameUserAsks {
				stocks[bidOrder.StockId].asks.Push(sua, sua.Price, sua.StockQuantity)
			}
		}()

		if topAskOrder == nil {
			l.Debugf("No matching top ask order currently. Adding to orderbook")
			stocks[bidOrder.StockId].bids.Push(bidOrder, bidOrder.Price, bidOrder.StockQuantity)
			depth.AddOrder(bidOrder.OrderType == Market, false, bidOrder.Price, bidOrder.StockQuantity)
			return true
		}

		l.Debugf("TopAskOrder found as: %+v", topAskOrder)
		l.Infof("Acquiring lock in order of User Ids")

		var firstUserId uint32
		var secondUserId uint32
		var isAskFirst bool

		//look out for error!!!
		if bidOrder.UserId < topAskOrder.UserId {
			firstUserId = bidOrder.UserId
			secondUserId = topAskOrder.UserId
			isAskFirst = false
		} else {
			firstUserId = topAskOrder.UserId
			secondUserId = bidOrder.UserId
			isAskFirst = true
		}

		firstLockChan, firstUser, err := getUser(firstUserId)
		if err != nil {
			l.Errorf("Errored: %+v", err)
			return false
		}
		defer close(firstLockChan)

		secondLockChan, secondUser, err := getUser(secondUserId)
		if err != nil {
			l.Errorf("Errored: %+v", err)
			return false
		}
		defer close(secondLockChan)

		l.Debugf("Acquired the locks on users %d and %d", firstUserId, secondUserId)

		if !isBidOrderMatching(bidOrder) {
			stockQuantityYetToBeFulfilled := bidOrder.StockQuantity - bidOrder.StockQuantityFulfilled
			l.Debugf("Unable to find the match. Putting bid in orderbook. Unfulfilled qty: %d", stockQuantityYetToBeFulfilled)
			if !bidOrder.IsClosed {
				stocks[bidOrder.StockId].bids.Push(bidOrder, bidOrder.Price, stockQuantityYetToBeFulfilled)
				depth.AddOrder(bidOrder.OrderType == Market, false, bidOrder.Price, stockQuantityYetToBeFulfilled)
			}
			return true
		}

		var (
			askDone bool
			bidDone bool
			tr      *Transaction
		)

		l.Debugf("Performing OrderFill transaction")

		//PerformOrderFillTransaction should update StockQuantityFulfilled and IsClosed
		if isAskFirst {
			askDone, bidDone, tr = PerformOrderFillTransaction(firstUser, secondUser, topAskOrder, bidOrder)
		} else {
			askDone, bidDone, tr = PerformOrderFillTransaction(secondUser, firstUser, topAskOrder, bidOrder)
		}

		if tr != nil {
			l.Infof("Trade made between bid and ask (%+v)", topAskOrder)

			depth.Trade(tr.Price, uint32(-tr.StockQuantity), tr.CreatedAt)
			depth.CloseOrder(topAskOrder.OrderType == Market, true, topAskOrder.Price, uint32(-tr.StockQuantity))
			// don't depth.CloseOrder( bidOrder) - It's not even added to depth yet

			// Trigger all the stoplosses that can be triggered.
			triggerStoplosses(tr)
		} else {
			l.Infof("Trade not made. AskDone = %+v, BidDone = %v", askDone, bidDone)

			if askDone {
				depth.CloseOrder(topAskOrder.OrderType == Market, true, topAskOrder.Price, topAskOrder.StockQuantity-topAskOrder.StockQuantityFulfilled)
			}
			/*
				if bidDone {
					do nothing. Bid hasn't been added to the depth yet
				}
			*/
		}

		// if bid not done, do nothing.

		// if bidDone {
		// 	return true, nil
		// } else if askDone {
		// 	stocks[bidOrder.StockId].asks.Pop()
		// 	return false, nil
		// }
		if askDone {
			l.Debugf("Popping topAskOrder %+v", topAskOrder)
			stocks[bidOrder.StockId].asks.Pop()
		}
		if bidDone {
			return true
		} else {
			return false
		}

		l.Errorf("PerformOrderFillTransaction returned both askDone, bidDone")
		return true
	}

	var (
		askDoneForNow bool
		bidDoneForNow bool
	)

	// Clear all existing orders in the book
	var existingAsks []*Ask
	for stock.asks.Size() != 0 {
		existingAsks = append(existingAsks, stock.asks.Pop())
	}

	for _, topAskOrder := range existingAsks {
		for !processAsk(topAskOrder) {

		}
		topAskOrder = stock.asks.Pop()
	}

	//run infinite loop
	for {
		select {
		case askOrder := <-stock.askChan:
			l.Debugf("Got ask %+v. Processing", askOrder)
			if askOrder.OrderType == StopLoss {
				l.Debugf("Adding stopLoss to the list")
				stocks[askOrder.StockId].askStoploss.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
				break
			}
			for {
				/*if askOrder.Type == StopLoss {
					stocks[askOrder.StockId].sellStoploss.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
				}*/

				askDoneForNow = processAsk(askOrder)
				if askDoneForNow {
					break
				}
				// processAsk returns true when it's done with this ask.
			}

		case bidOrder := <-stock.bidChan:
			l.Debugf("Got bid %+v. Processing", bidOrder)
			if bidOrder.OrderType == StopLoss {
				l.Debugf("Adding stopLoss to the list")
				stocks[bidOrder.StockId].bidStoploss.Push(bidOrder, bidOrder.Price, bidOrder.StockQuantity)
				break
			}
			for {
				bidDoneForNow = processBid(bidOrder)
				if bidDoneForNow {
					break
				}
				// processBid returns true when it's done with this bid.
			}
		}
	}
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
		stocks[stockId] = StockDetails{
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
			stocks[openAskOrder.StockId].askStoploss.Push(openAskOrder, openAskOrder.Price, 0)
		} else {
			askUnfulfilledQuantity = openAskOrder.StockQuantity - openAskOrder.StockQuantityFulfilled
			stocks[openAskOrder.StockId].asks.Push(openAskOrder, openAskOrder.Price, askUnfulfilledQuantity)
		}
	}

	//Load open bid orders into priority queue
	for _, openBidOrder := range openBidOrders {
		if openBidOrder.OrderType == StopLoss {
			stocks[openBidOrder.StockId].bids.Push(openBidOrder, openBidOrder.Price, 0)
		} else {
			bidUnfulfilledQuantity = openBidOrder.StockQuantity - openBidOrder.StockQuantityFulfilled
			stocks[openBidOrder.StockId].bids.Push(openBidOrder, openBidOrder.Price, bidUnfulfilledQuantity)
		}
	}

	for _, stockId := range stockIds {
		go StartStockMatching(stocks[stockId], stockId)
	}
}

func min(i, j uint32) uint32 {
	if i < j {
		return i
	} else {
		return j
	}
}

/*
*	function to check if the placed askorder price is less than that of the highest bidder for that stock
 */
func isAskOrderMatching(askOrder *Ask) bool {
	stockId := askOrder.StockId
	maxBid := stocks[stockId].bids.Head()
	if maxBid == nil {
		return false
	}
	if askOrder.OrderType == Market {
		return true
	}
	return maxBid.Price >= askOrder.Price
}

/*
*	function to check if the placed bidorder price is greater than that of the lowest askorder for that stock
 */
func isBidOrderMatching(bidOrder *Bid) bool {
	stockId := bidOrder.StockId
	minAsk := stocks[stockId].asks.Head()
	if minAsk == nil {
		return false
	}
	if bidOrder.OrderType == Market {
		return true
	}
	return minAsk.Price <= bidOrder.Price
}
