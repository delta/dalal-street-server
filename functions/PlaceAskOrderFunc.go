package functions

import (
	"fmt"
)

var maxQuantityLimit uint32

func PlaceAskOrder(request PlaceAskOrderRequest) err {
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
	return err
}
