package models

import (
	"container/list"

	"github.com/Sirupsen/logrus"
)

type Trade struct {
	TradeQuantity uint32
	TradePrice    uint32
	TradeTime     string
}

type MarketDepth struct {
	AskDepth     map[uint32]uint32
	BidDepth     map[uint32]uint32
	LatestTrades *list.List
}

func NewMarketDepth() *MarketDepth {
	return &MarketDepth{
		AskDepth:     make(map[uint32]uint32),
		BidDepth:     make(map[uint32]uint32),
		LatestTrades: list.New(),
	}
}

func (md *MarketDepth) AddOrder(isAsk bool, price uint32, stockQuantity uint32) {
	depth := md.AskDepth
	if !isAsk {
		depth = md.BidDepth
	}

	depth[price] += stockQuantity
}

func (md *MarketDepth) Trade(price, qty uint32, createdAt string) {
	t := &Trade{
		qty,
		price,
		createdAt,
	}

	if md.LatestTrades.Len() >= 10 {
		f := md.LatestTrades.Front()
		md.LatestTrades.Remove(f)
	}

	md.LatestTrades.PushBack(t)
}

func (md *MarketDepth) CloseOrder(isAsk bool, price uint32, stockQuantity uint32) {
	depth := md.AskDepth
	if !isAsk {
		depth = md.BidDepth
	}

	depth[price] += stockQuantity
}

type StockDetails struct {
	askChan     chan *Ask
	bidChan     chan *Bid
	asks        *AskPQueue
	bids        *BidPQueue
	askStoploss *AskPQueue
	bidStoploss *BidPQueue
	depth       MarketDepth
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
		topBidOrder := stocks[askOrder.StockId].bids.Head()
		depth := stocks[askOrder.StockId].depth

		if topBidOrder == nil {
			stocks[askOrder.StockId].asks.Push(askOrder, askOrder.Price, askOrder.StockQuantity)
			depth.AddOrder(true, askOrder.Price, askOrder.StockQuantity)
			return true
		}

		l.Infof("Acquiring lock in order of User Ids")

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

		l.Debugf("Acquired locks")

		if !isAskOrderMatching(askOrder) {
			l.Debugf("Didn't find match. Pushing!")
			stockQuantityYetToBeFulfilled := askOrder.StockQuantity - askOrder.StockQuantityFulfilled
			if !askOrder.IsClosed {
				stocks[askOrder.StockId].asks.Push(askOrder, askOrder.Price, stockQuantityYetToBeFulfilled)
				depth.AddOrder(true, askOrder.Price, stockQuantityYetToBeFulfilled)
			}
			return true
		}

		l.Debugf("Perform OrderFill Transaction")

		var (
			askDone bool
			bidDone bool
			tr      *Transaction
		)
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
			// tr is always AskTransaction. So its StockQty < 0. Make it positive.
			depth.Trade(tr.Price, uint32(-tr.StockQuantity), tr.CreatedAt)
			//depth.CloseOrder(true, askOrder.Price, tr.StockQuantity) - don't! Haven't added ask to depth
			depth.CloseOrder(false, topBidOrder.Price, uint32(-tr.StockQuantity))
		} else {
			/*
				if askDone {
					 do nothing. We haven't even added the ask to the depth
				}
			*/
			if bidDone {
				depth.CloseOrder(false, topBidOrder.Price, topBidOrder.StockQuantity-topBidOrder.StockQuantityFulfilled)
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
		topAskOrder := stocks[bidOrder.StockId].asks.Head()
		depth := stocks[bidOrder.StockId].depth

		if topAskOrder == nil {
			stocks[bidOrder.StockId].bids.Push(bidOrder, bidOrder.Price, bidOrder.StockQuantity)
			depth.AddOrder(false, bidOrder.Price, bidOrder.StockQuantity)
			return true
		}

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

		l.Debugf("Acquired")

		if !isBidOrderMatching(bidOrder) {
			l.Debugf("Order not matching even in first attempt. Pushing!")
			stockQuantityYetToBeFulfilled := bidOrder.StockQuantity - bidOrder.StockQuantityFulfilled
			if !bidOrder.IsClosed {
				stocks[bidOrder.StockId].bids.Push(bidOrder, bidOrder.Price, stockQuantityYetToBeFulfilled)
				depth.AddOrder(false, bidOrder.Price, stockQuantityYetToBeFulfilled)
			}
			return true
		}

		var (
			askDone bool
			bidDone bool
			tr      *Transaction
		)

		l.Debugf("Performing OrderFillTransaction")

		//PerformOrderFillTransaction should update StockQuantityFulfilled and IsClosed
		if isAskFirst {
			askDone, bidDone, tr = PerformOrderFillTransaction(firstUser, secondUser, topAskOrder, bidOrder)
		} else {
			askDone, bidDone, tr = PerformOrderFillTransaction(secondUser, firstUser, topAskOrder, bidOrder)
		}

		if tr != nil {
			depth.Trade(tr.Price, uint32(-tr.StockQuantity), tr.CreatedAt)
			depth.CloseOrder(true, topAskOrder.Price, uint32(-tr.StockQuantity))
			// don't depth.CloseOrder( bidOrder) - It's not even added to depth yet
		} else {
			if askDone {
				depth.CloseOrder(true, topAskOrder.Price, topAskOrder.StockQuantity-topAskOrder.StockQuantityFulfilled)
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
	//run infinite loop
	for {
		select {
		case askOrder := <-stock.askChan:
			l.Debugf("Got ask. Processing")
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
			l.Debugf("Got bid. Processing")
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

	db, err := DbOpen()
	if err != nil {
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
		panic("Failed to load stock ids in matching engine")
	}

	//Load open ask orders from database
	if err := db.Where("isClosed = ?", 0).Find(&openAskOrders).Error; err != nil {
		panic("Error loading open ask orders in matching engine")
	}

	//Load open bid orders from database
	if err := db.Where("isClosed = ?", 0).Find(&openBidOrders).Error; err != nil {
		panic("Error loading open bid orders in matching engine")
	}

	for _, stockId := range stockIds {
		stocks[stockId] = StockDetails{
			askChan: make(chan *Ask),
			bidChan: make(chan *Bid),
			asks:    NewAskPQueue(MINPQ), //lower price has higher priority
			bids:    NewBidPQueue(MAXPQ), //higher price has higher priority
		}
	}

	//Load open ask orders into priority queue
	for _, openAskOrder := range openAskOrders {
		askUnfulfilledQuantity = openAskOrder.StockQuantity - openAskOrder.StockQuantityFulfilled
		stocks[openAskOrder.StockId].asks.Push(openAskOrder, openAskOrder.Price, askUnfulfilledQuantity)
	}

	//Load open bid orders into priority queue
	for _, openBidOrder := range openBidOrders {
		bidUnfulfilledQuantity = openBidOrder.StockQuantity - openBidOrder.StockQuantityFulfilled
		stocks[openBidOrder.StockId].bids.Push(openBidOrder, openBidOrder.Price, bidUnfulfilledQuantity)
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
	return maxBid.Price > askOrder.Price
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
	return minAsk.Price < bidOrder.Price
}
