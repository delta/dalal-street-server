package models

import (
	"github.com/Sirupsen/logrus"
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

// GetOrderDepositTransactionRef returns a reference of
func GetOrderDepositTransactionRef(transactionID, orderID uint32, isAsk bool) *OrderDepositTransaction {
	return &OrderDepositTransaction{
		TransactionId: transactionID,
		OrderId:       orderID,
		IsAsk:         isAsk,
		CreatedAt:     utils.GetCurrentTimeISO8601(),
	}
}

// GetPlaceOrderTransactionDetails returns price and quantity at which PlaceOrderTransaction was created
func GetPlaceOrderTransactionDetails(orderID uint32, isAsk bool) (int64, int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetPlaceOrderTransactionDetails",
		"userID": orderID,
	})

	db := getDB()
	sql := "SELECT transactionId FROM OrderDepositTransactions WHERE orderID = ? and isAsk = ?"
	rows, err := db.Raw(sql, orderID, isAsk).Rows()
	if err != nil {
		l.Errorf("Error retrieving transactionId. Error: %+v", err)
		return 0, 0, err
	}

	defer rows.Close()
	if !rows.Next() {
		return 0, 0, InvalidOrderIDError{}
	}

	var transactionId uint32
	rows.Scan(&transactionId)

	l.Infof("Retrieving reserved asset for transactionId %d", transactionId)

	sql = "SELECT total, stockQuantity from Transactions WHERE id = ?"
	trows, err := db.Raw(sql, transactionId).Rows()

	if err != nil {
		l.Errorf("Error while retrieving reserve details. Error: %+v", err)
		return 0, 0, err
	}

	defer trows.Close()
	if !trows.Next() {
		return 0, 0, InvalidTransaction{}
	}

	var total int64
	var stockQuantity int64
	trows.Scan(&total, &stockQuantity)

	l.Infof("Retrieved reserved asset. Cash reserved %d and Stock Reserved %d for transaction %d", total, stockQuantity, transactionId)

	return -total, stockQuantity, nil
}
