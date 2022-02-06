package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Short sell bank is an abstraction of inventory, holds
// the number of stocks of stock available for lending (which is further used for shorting by the user)
type ShortSellBank struct {
	StockId         uint32 `gorm:"column:stockId;primary_key;" json:"stockId"`
	AvailableStocks uint32 `gorm:"column:availableStocks; default 0;" json:"availableStocks"`
}

// short sell lends stores all the stock lends to a user
// this will be used to square-off at EOD
type ShortSellLends struct {
	StockId       uint32 `gorm:"column:stockId;" json:"stockId"`
	UserId        uint32 `gorm:"column:userId;" json:"userId"`
	StockQuantity uint32 `gorm:"column:stockQuantity;" json:"stockQuantity"`
	IsSquaredOff  bool   `gorm:"column:isSquaredOff; default false;" json:"isSquaredOff"`
}

func (ShortSellBank) TableName() string {
	return "ShortSellBank"
}

func (ShortSellLends) TableName() string {
	return "ShortSellLends"
}

func getAvailableLendStocks(stockId uint32) (uint32, error) {
	l := logger.WithFields(logrus.Fields{
		"method":  "getAvailableLendStocks",
		"stockId": stockId,
	})

	l.Info("fetching available stocks to lend from db")

	db := getDB()

	var shortSellBank ShortSellBank

	if err := db.Find(&shortSellBank, "stockId = ?", stockId).Select("availableStocks").Error; err != nil {
		l.Errorf("error fetching available stocks to lend from db %+v", err)
		return shortSellBank.AvailableStocks, err
	}

	return shortSellBank.AvailableStocks, nil
}

func saveLendStockTransaction(lendStocksTransaction *Transaction, tx *gorm.DB) error {
	// subtracting available stocks before saving lending transaction
	if err := updateShortSellBank(lendStocksTransaction.StockId, -uint32(lendStocksTransaction.StockQuantity), tx); err != nil {
		return err
	}

	// creating transaction
	if err := tx.Save(lendStocksTransaction).Error; err != nil {
		return err
	}

	// saving lend
	if err := createShortSellLend(lendStocksTransaction.StockId, lendStocksTransaction.UserId, uint32(lendStocksTransaction.StockQuantity), tx); err != nil {
		return err
	}

	return nil
}

func updateShortSellBank(stockId uint32, stockQuantity uint32, tx *gorm.DB) error {

	availableStocks, err := getAvailableLendStocks(stockId)

	if err != nil {
		return err
	}

	ssb := &ShortSellBank{
		StockId:         stockId,
		AvailableStocks: availableStocks + stockQuantity,
	}

	// updating available stocks for lending in db
	if err := tx.Save(ssb).Error; err != nil {
		return err
	}

	return nil
}

func createShortSellLend(stockId, userId, stockQuantity uint32, tx *gorm.DB) error {
	fmt.Println("userId", userId)
	l := logger.WithFields(logrus.Fields{
		"method":  "getAvailableLendStocks",
		"stockId": stockId,
	})

	l.Debugf("attemting to save short sell lend of %d quantity", userId)

	ssl := &ShortSellLends{
		StockId:       stockId,
		UserId:        userId,
		StockQuantity: stockQuantity,
		IsSquaredOff:  false,
	}

	if err := tx.Create(ssl).Error; err != nil {
		l.Errorf("error creating short sell lends %+v", err)
		return err
	}

	return nil
}
