package functions

import (
	"fmt"
)

//TODO: Change constant
var maxQuantityLimit uint32

func PlaceBidOrder(request PlaceBidOrderRequest) err {

	mySession := session.GetSession(request.GetSessionId())
	userId := mySession.get("userId")
	stockId := request.GetStockId()
	orderType := request.GetOrderType()
	price := request.GetPrice()
	quantity := request.GetStockQuantity()

	db, err := dbConn()
	if err != nil {
		return err
	}

	var user User
	db.First(&user, userId)

	err = CheckLimits(quantity)
	if err != nil {
		return err
	}

	var bid Bid
	bid := db.First(&bid, "StockId=?", stockId)

	if bid != nil {
		err = CheckCashForExistingBid(bid, user)
		if err != nil {
			return err
		}
	} else {
		err = CheckCash(price, quantity, user)
		if err != nil {
			return err
		}
		//TODO: Call the function for matching using the engine
		return err
	}
}

func CheckLimits(quantity uint32) err {
	if quantity <= 0 {
		return bidErr("invalid quantity")
	}
	if quantity > maxQuantityLimit {
		return bidErr("quantity limit exceeded")
	}
	return nil
}

func CheckCashForExistingBid(bid Bid, user User) err {
	if user.Cash < (bid.StockQuanity - bid.StockQuantityFullfilled) {
		return bidErr("insufficient cash")
	}
	return nil
}

func CheckCash(price uint32, quantity uint32, user User) err {
	if user.Cash < (price * quantity) {
		return bidErr("insufficient cash")
	}
	return nil
}

type bidErr struct {
	s string
}

func (e *bidErr) Error() string {
	return e.s
}
