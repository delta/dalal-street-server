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
 */
func getTradePriceAndQty(ask *models.Ask, bid *models.Bid) (uint64, uint64) {
	var bidUnfulfilledStockQuantity = bid.StockQuantity - bid.StockQuantityFulfilled
	var askUnfulfilledStockQuantity = ask.StockQuantity - ask.StockQuantityFulfilled

	var stockTradeQty = utils.MinInt64(askUnfulfilledStockQuantity, bidUnfulfilledStockQuantity)
	var stockTradePrice uint64

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
