package matchingengine

import (
	"github.com/delta/dalal-street-server/models"
	"github.com/delta/dalal-street-server/utils"
)

/*
 *	Helper function to check if it's a Market order
 */
func isMarket(oType models.OrderType) bool {
	return oType == models.Market || oType == models.StopLossActive
}

/*
 *	Helper function to check if a transaction is possible
 */
func isOrderMatching(askTop *models.Ask, bidTop *models.Bid) bool {
	if isMarket(bidTop.OrderType) || isMarket(askTop.OrderType) {
		return true
	}
	return bidTop.Price >= askTop.Price
}

/*
 * getTradePriceAndQty returns the trade price and quantity for a particular match of orders
 To determine quantity : qty = min(bidUnfulfilledStockQuantity, askUnfulfilledStockQuantity)
 To determine price :
 1. If both ask and bid are market orders, trade price is the current market price
 2. If one of them is market but other is limit, trade price is the price of the limit order
 3. If neither are market, trade price is the price of the order that was placed first
*/
func getTradePriceAndQty(ask *models.Ask, bid *models.Bid) (uint32, uint32) {
	var bidUnfulfilledStockQuantity = bid.StockQuantity - bid.StockQuantityFulfilled
	var askUnfulfilledStockQuantity = ask.StockQuantity - ask.StockQuantityFulfilled

	var stockTradeQty = utils.MinInt(askUnfulfilledStockQuantity, bidUnfulfilledStockQuantity)
	var stockTradePrice uint32

	//set transaction price based on order type
	if isMarket(ask.OrderType) && isMarket(bid.OrderType) {
		stock, _ := models.GetStockCopy(ask.StockId)
		stockTradePrice = stock.CurrentPrice
	} else if isMarket(ask.OrderType) {
		stockTradePrice = bid.Price
	} else if isMarket(bid.OrderType) {
		stockTradePrice = ask.Price
	} else if ask.CreatedAt < bid.CreatedAt {
		stockTradePrice = ask.Price
	} else {
		stockTradePrice = bid.Price
	}

	return stockTradePrice, stockTradeQty
}
