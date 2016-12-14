package models

//enum for tables Asks and Bids
type OrderType uint8

const (
	Limit OrderType = iota
	Market
	Stoploss
)

var orderTypes = [...]string{
	"Limit",
	"Market",
	"Stoploss",
}

func (ot OrderType) String() string {
	return orderTypes[ot-1]
}

//enum for table Transactions
type TransactionType uint8

const (
	FromExchange TransactionType = iota
	OrderFill
	Mortgage
	Dividend
)

var transactionTypes = [...]string{
	"FromExchange",
	"OrderFill",
	"Mortgage",
	"Dividend",
}

func (trType TransactionType) String() string {
	return transactionTypes[trType-1]
}