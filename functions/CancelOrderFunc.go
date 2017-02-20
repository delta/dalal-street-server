package functions

import (
	"fmt"
)

func CancelOrder(request PlaceBidOrderRequest) err {
	db, err := dbConn()
	return err
}
