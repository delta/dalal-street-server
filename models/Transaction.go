package models

type TransactionType uint8

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
	return transactionTypes[trType-1]
}

type Transaction struct {
	Id            uint32          `gorm:"primary_key;AUTO_INCREMENT"`
	UserId        uint32          `gorm:"column:userId;not null"`
	StockId       uint32          `gorm:"column:stockId;not null"`
	Type          TransactionType `gorm:"column:type;not null"`
	StockQuantity uint32          `gorm:"column:stockQuantity;not null"`
	Price         uint32          `gorm:"not null"`
	CreatedAt     string          `gorm:"column:createdAt;not null"`
}

func (Transaction) TableName() string {
	return "Transactions"
}