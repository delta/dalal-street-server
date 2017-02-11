package models

import (
	"database/sql/driver"
	"fmt"

	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type TransactionType uint8

func (tt *TransactionType) Scan(value interface{}) error {
	switch string(value.([]byte)) {
	case "FromExchangeTransaction":
		*tt = 0
	case "OrderFillTransaction":
		*tt = 1
	case "MortgageTransaction":
		*tt = 2
	case "DividendTransaction":
		*tt = 3
	default:
		return fmt.Errorf("Invalid value for TransactionType. Got %s", string(value.([]byte)))
	}
	return nil
}

func (tt TransactionType) Value() (driver.Value, error) { return tt.String(), nil }

const (
	FromExchangeTransaction TransactionType = iota
	OrderFillTransaction
	MortgageTransaction
	DividendTransaction
)

var transactionTypes = [...]string{
	"FromExchangeTransaction",
	"OrderFillTransaction",
	"MortgageTransaction",
	"DividendTransaction",
}

func (trType TransactionType) String() string {
	return transactionTypes[trType]
}

type Transaction struct {
	Id            uint32          `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId        uint32          `gorm:"column:userId;not null" json:"user_id"`
	StockId       uint32          `gorm:"column:stockId;not null" json:"stock_id"`
	Type          TransactionType `gorm:"column:type;not null" json:"type"`
	StockQuantity int32           `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	Price         uint32          `gorm:"not null" json:"price"`
	Total         int32           `gorm:"not null" json:"total"`
	CreatedAt     string          `gorm:"column:createdAt;not null" json:"created_at"`
}

func (Transaction) TableName() string {
	return "Transactions"
}

func (t *Transaction) ToProto() *models_proto.Transaction {
	pTrans := &models_proto.Transaction{
		Id:      t.Id,
		UserId:  t.UserId,
		StockId: t.StockId,
		// Type: t.Type,
		StockQuantity: t.StockQuantity,
		Price:         t.Price,
		Total:         t.Total,
		CreatedAt:     t.CreatedAt,
	}

	if t.Type == FromExchangeTransaction {
		pTrans.Type = models_proto.TransactionType_FROM_EXCHANGE_TRANSACTION
	} else if t.Type == OrderFillTransaction {
		pTrans.Type = models_proto.TransactionType_ORDER_FILL_TRANSACTION
	} else if t.Type == MortgageTransaction {
		pTrans.Type = models_proto.TransactionType_MORTGAGE_TRANSACTION
	} else if t.Type == DividendTransaction {
		pTrans.Type = models_proto.TransactionType_DIVIDEND_TRANSACTION
	}

	return pTrans
}
