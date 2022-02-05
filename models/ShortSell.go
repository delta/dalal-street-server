package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Short sell bank is an abstraction of inventory, holds
// the number of stocks of stock available for lending (which is further used for shorting by the user)

type ShortSellBank struct {
	StockId         uint32 `gorm:"column:stockId;primary_key; json:"stockId"`
	AvailableStocks uint32 `gorm:"column:availableStocks; default 0; json:"availableStocks"`
}

func (ShortSellBank) TableName() string {
	return "ShortSellBank"
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

	fmt.Println(lendStocksTransaction.Type)

	if err := tx.Save(lendStocksTransaction).Error; err != nil {
		return err
	}

	//TODO save this transaction in some table for later squaring off

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
