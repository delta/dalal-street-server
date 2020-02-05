package models

import (
	"github.com/Sirupsen/logrus"
)

func BankruptStock(stockID uint32, isBankrupt bool) error {
	var l = logger.WithFields(logrus.Fields{
		"method":     "GetMyClosedAsks",
		"stockID":    stockID,
		"isBankrupt": isBankrupt,
	})

	l.Infof("Attempting to Bankrupt stock")

	allStocks.Lock()
	stockNLock, ok := allStocks.m[stockID]
	if !ok {
		return InvalidStockError
	}
	allStocks.Unlock()

	stockNLock.Lock()
	defer stockNLock.Unlock()

	stock := stockNLock.stock
	oldStockCopy := *stock

	stock.IsBankrupt = isBankrupt

	db := getDB()

	if err := db.Save(stock).Error; err != nil {
		*stock = oldStockCopy
		return err
	}

	return nil
}
