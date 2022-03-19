package models

import (
	"database/sql/driver"
	"fmt"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
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
	case "OrderFeeTransaction":
		*tt = 4
	case "TaxTransaction":
		*tt = 5
	case "PlaceOrderTransaction":
		*tt = 6
	case "CancelOrderTransaction":
		*tt = 7
	case "ReserveUpdateTransaction":
		*tt = 8
	case "ShortSellTransaction":
		*tt = 9
	case "IpoAllotmentTransaction":
		*tt = 10
	default:
		return fmt.Errorf("invalid value for TransactionType. Got %s", string(value.([]byte)))
	}
	return nil
}

func (tt TransactionType) Value() (driver.Value, error) { return tt.String(), nil }

const (
	FromExchangeTransaction TransactionType = iota
	OrderFillTransaction
	MortgageTransaction
	DividendTransaction
	OrderFeeTransaction
	TaxTransaction
	PlaceOrderTransaction
	CancelOrderTransaction
	ReserveUpdateTransaction
	ShortSellTransaction
	IpoAllotmentTransaction
)

var transactionTypes = [...]string{
	"FromExchangeTransaction",
	"OrderFillTransaction",
	"MortgageTransaction",
	"DividendTransaction",
	"OrderFeeTransaction",
	"TaxTransaction",
	"PlaceOrderTransaction",
	"CancelOrderTransaction",
	"ReserveUpdateTransaction",
	"ShortSellTransaction",
	"IpoAllotmentTransaction",
}

func (trType TransactionType) String() string {
	return transactionTypes[trType]
}

type Transaction struct {
	Id                    uint32          `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId                uint32          `gorm:"column:userId;not null" json:"user_id"`
	StockId               uint32          `gorm:"column:stockId" json:"stock_id"`
	Type                  TransactionType `gorm:"column:type;not null" json:"type"`
	ReservedStockQuantity int64           `gorm:"column:reservedStockQuantity;not null" json:"reserved_stock_quantity"`
	StockQuantity         int64           `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	Price                 uint64          `gorm:"not null" json:"price"`
	ReservedCashTotal     int64           `gorm:"column:reservedCashTotal;not null" json:"reserved_cash_total"`
	Total                 int64           `gorm:"not null" json:"total"`
	CreatedAt             string          `gorm:"column:createdAt;not null" json:"created_at"`
}

func (Transaction) TableName() string {
	return "Transactions"
}

func (t *Transaction) ToProto() *models_pb.Transaction {
	pTrans := &models_pb.Transaction{
		Id:      t.Id,
		UserId:  t.UserId,
		StockId: t.StockId,
		// Type: t.Type,
		ReservedStockQuantity: t.ReservedStockQuantity,
		StockQuantity:         t.StockQuantity,
		Price:                 t.Price,
		ReservedCashTotal:     t.ReservedCashTotal,
		Total:                 t.Total,
		CreatedAt:             t.CreatedAt,
	}

	if t.Type == FromExchangeTransaction {
		pTrans.Type = models_pb.TransactionType_FROM_EXCHANGE_TRANSACTION
	} else if t.Type == OrderFillTransaction {
		pTrans.Type = models_pb.TransactionType_ORDER_FILL_TRANSACTION
	} else if t.Type == MortgageTransaction {
		pTrans.Type = models_pb.TransactionType_MORTGAGE_TRANSACTION
	} else if t.Type == DividendTransaction {
		pTrans.Type = models_pb.TransactionType_DIVIDEND_TRANSACTION
	} else if t.Type == OrderFeeTransaction {
		pTrans.Type = models_pb.TransactionType_ORDER_FEE_TRANSACTION
	} else if t.Type == TaxTransaction {
		pTrans.Type = models_pb.TransactionType_TAX_TRANSACTION
	} else if t.Type == PlaceOrderTransaction {
		pTrans.Type = models_pb.TransactionType_PLACE_ORDER_TRANSACTION
	} else if t.Type == CancelOrderTransaction {
		pTrans.Type = models_pb.TransactionType_CANCEL_ORDER_TRANSACTION
	} else if t.Type == ReserveUpdateTransaction {
		pTrans.Type = models_pb.TransactionType_RESERVE_UPDATE_TRANSACTION
	} else if t.Type == ShortSellTransaction {
		pTrans.Type = models_pb.TransactionType_SHORT_SELL_TRANSACTION
	} else if t.Type == IpoAllotmentTransaction {
		pTrans.Type = models_pb.TransactionType_IPO_ALLOTMENT_TRANSACTION
	}

	return pTrans
}

func GetTransactions(userId, lastId, count uint32) (bool, []*Transaction, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetTransactions",
		"userId": userId,
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get transactions")

	db := getDB()

	var transactions []*Transaction

	//set default value of count if it is zero
	if count == 0 {
		count = GET_TRANSACTION_COUNT
	} else {
		count = utils.MinInt32(count, GET_TRANSACTION_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	if err := db.Where("userId = ?", userId).Order("id desc").Limit(count).Find(&transactions).Error; err != nil {
		return true, nil, err
	}

	var moreExists = len(transactions) >= int(count)
	l.Infof("Successfully fetched transactions")
	return moreExists, transactions, nil
}

func GetAskTransactionsForStock(stockID, count uint32) ([]*Transaction, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetTransactionsForStock",
		"count":  count,
	})

	l.Debugf("Attempting")

	db := getDB()

	var transactions []*Transaction

	//get latest events if lastId is zero
	db = db.Where("stockID = ?", stockID).Where("stockQuantity < 0").Where("`type` = ?", "OrderFillTransaction")
	if err := db.Order("id desc").Limit(count).Find(&transactions).Error; err != nil {
		return nil, err
	}

	l.Debugf("Done")
	return transactions, nil
}

// GetTransactionRef creates and returns a reference of a Transaction
func GetTransactionRef(userID, stockID uint32, ttype TransactionType, reservedStockQuantity int64, stockQuantity int64, price uint64, reservedCashTotal int64, total int64) *Transaction {
	return &Transaction{
		UserId:                userID,
		StockId:               stockID,
		Type:                  ttype,
		ReservedStockQuantity: reservedStockQuantity,
		StockQuantity:         stockQuantity,
		Price:                 price,
		ReservedCashTotal:     reservedCashTotal,
		Total:                 total,
		CreatedAt:             utils.GetCurrentTimeISO8601(),
	}
}
