package models

import (
	"github.com/sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils"
)

type OrderDepositTransaction struct {
	TransactionId uint32 `gorm:"column:transactionId;not null" json:"transaction_id"`
	OrderId       uint32 `gorm:"column:orderId;not null" json:"bid_id"`
	IsAsk         bool   `gorm:"column:isAsk;not null" json:"is_ask"`
	CreatedAt     string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (OrderDepositTransaction) TableName() string {
	return "OrderDepositTransactions"
}

// makeOrderDepositTransactionRef returns a reference of OrderDepositTransaction
func makeOrderDepositTransactionRef(transactionID, orderID uint32, isAsk bool) *OrderDepositTransaction {
	return &OrderDepositTransaction{
		TransactionId: transactionID,
		OrderId:       orderID,
		IsAsk:         isAsk,
		CreatedAt:     utils.GetCurrentTimeISO8601(),
	}
}

// getPlaceOrderTransactionDetails returns price and quantity at which PlaceOrderTransaction was created
func getPlaceOrderTransactionDetails(orderID uint32, isAsk bool) (int64, int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "getPlaceOrderTransactionDetails",
		"userID": orderID,
	})

	db := getDB()

	var totalPrice int64
	var stocksInBank int64

	sql := "Select tx.total as totalPrice, tx.stockQuantity as stocksInBank from Transactions tx INNER JOIN OrderDepositTransactions odtx on tx.id = odtx.transactionID WHERE odtx.orderID = ? and isAsk = ?"
	rows, err := db.Raw(sql, orderID, isAsk).Rows()
	if err != nil {
		l.Errorf("Error retrieving transactionId. Error: %+v", err)
		return 0, 0, err
	}

	defer rows.Close()
	if !rows.Next() {
		return 0, 0, InvalidOrderIDError{}
	}

	rows.Scan(&totalPrice, &stocksInBank)

	l.Infof("Retrieved reserved asset. Cash reserved %d and Stock Reserved %d for order %d %d", totalPrice, stocksInBank, orderID, isAsk)

	return -totalPrice, stocksInBank, nil
}
