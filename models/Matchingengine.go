package models

import (
	_ "github.com/Sirupsen/logrus"
)

//type of channel where ask orders will be streamed
type askChanStruct struct {
	order   *Ask
	matcher func(*Ask, *Bid) (bool, bool)
}

//type of channel where bid orders will be streamed
type bidChanStruct struct {
	order   *Bid
	matcher func(*Ask, *Bid) (bool, bool)
}

type StockDetails struct {
	askChan chan askChanStruct
	bidChan chan bidChanStruct
	asks    *AskPQueue
	bids    *BidPQueue
}

//map to store details of the placed orders. Maps stockId to the PlacedOrderDetails for that stockId
var stocks map[uint32]StockDetails

func AddAskOrder(askOrder *Ask, matcher func(*Ask, *Bid) (bool, bool)) {
	stocks[askOrder.StockId].askChan <- askChanStruct{askOrder, matcher}
}

func AddBidOrder(bidOrder *Bid, matcher func(*Ask, *Bid) (bool, bool)) {
	stocks[bidOrder.StockId].bidChan <- bidChanStruct{bidOrder, matcher}
}

//primary function to perform matching and transaction(through callback)
func StartStockMatching(stock StockDetails) {
	//run infinite loop
	for {
		select {
		case askOrder := <-stock.askChan:
			go func() {
				for {
					//askUserLock := getUser(askOrder.order.UserId)
					//if askOrder.order.IsClosed || !isAskOrderMatching(askOrder.order) then break
					//
					//bidUserLock := getUser(bid...)
					//// userid order.
					//
					topBidOrder := stocks[askOrder.order.StockId].bids.Head()
					//matcher should update StockQuantityFulfilled and IsClosed
					askFaulty, bidFaulty := askOrder.matcher(askOrder.order, topBidOrder)
					if askFaulty {
						break
					} else if bidFaulty || topBidOrder.IsClosed {
						stocks[askOrder.order.StockId].bids.Pop()
					}

					//unlock
				}

				stockQuantityYetToBeFulfilled := askOrder.order.StockQuantity - askOrder.order.StockQuantityFulfilled
				if !askOrder.order.IsClosed {
					stocks[askOrder.order.StockId].asks.Push(askOrder.order, askOrder.order.Price, stockQuantityYetToBeFulfilled)
				}
			}()
		case bidOrder := <-stock.bidChan:
			go func() {
				bidOrder.order.Lock()
				defer bidOrder.order.Unlock()
				for !bidOrder.order.IsClosed && isBidOrderMatching(bidOrder.order) {
					topAskOrder := stocks[bidOrder.order.StockId].asks.Head()
					//matcher should update StockQuantityFulfilled and isClosed
					askFaulty, bidFaulty := bidOrder.matcher(topAskOrder, bidOrder.order)
					if bidFaulty {
						break
					} else if askFaulty || topAskOrder.IsClosed {
						stocks[bidOrder.order.StockId].asks.Pop()
					}
				}

				stockQuantityYetToBeFulfilled := bidOrder.order.StockQuantity - bidOrder.order.StockQuantityFulfilled
				if !bidOrder.order.IsClosed {
					stocks[bidOrder.order.StockId].bids.Push(bidOrder.order, bidOrder.order.Price, stockQuantityYetToBeFulfilled)
				}
			}()
		}
	}
}

func Init(stockIds []uint32) {
	for _, stockId := range stockIds {
		stocks[stockId] = StockDetails{
			askChan: make(chan askChanStruct),
			bidChan: make(chan bidChanStruct),
			asks:    NewAskPQueue(MINPQ), //lower price has higher priority
			bids:    NewBidPQueue(MAXPQ), //higher price has higher priority
		}
		go StartStockMatching(stocks[stockId])
	}
}

func min(i, j uint32) uint32 {
	if i < j {
		return i
	} else {
		return j
	}
}

func isAskOrderMatching(askOrder *Ask) bool {
	stockId := askOrder.StockId
	maxBid := stocks[stockId].bids.Head()
	if maxBid == nil {
		return false
	}
	return maxBid.Price > askOrder.Price
}

func isBidOrderMatching(bidOrder *Bid) bool {
	stockId := bidOrder.StockId
	minAsk := stocks[stockId].asks.Head()
	if minAsk == nil {
		return false
	}
	return minAsk.Price < bidOrder.Price
}

/*


Stocks[
    askChan chan struct{ Ask, func(Ask, Bid) (AskFaulty, BidFaulty) }
    bidChan chan Bid

    Asks[]
    Bids[]
]

func AddAskOrder(a *Ask, notifyChan chan chan Bid) {
    stocks[a.StockId].askChan <- { ask, notify }
}

func AddBidOrder(b *Bid) {
    stocks[a.StockId].bidChan <- bid
}

func startStock(stock) {
    for {
        select {
        case x := <-stock.askChan:
            go func() {
                while( x.StocksFulFilled < x.StockQuantity && stock.Bids.top.matches ):
                    x.StocksFulfilled += min(x.StockQuantity, stock.Bids.StockQuantity)
                    askFaulty, bidFaulty := x.Match(x.Ask, stock.Bids.Top)
                    if askFaulty:
                        continue
                    else if bidFaulty or stock.Bids.Top.FulFilled:
                        stock.Bids.Pop()

                if x.unfulfilled:
                    stock.Asks.insert(x)
            }()
        case y := <-stock.BidChan:
            //blah
        }
    }
}


*/
