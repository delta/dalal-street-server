package models

import (
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
	Id            uint32 `gorm:"column:id;" json:"id"`
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

// returns available stocks to lend for a stock
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

// updates available stocks for lend, saves shortsell transaction and creates shortsell lend later squareOff
// utility func, called inside placeAskOrder func
func saveShortSellLendTransaction(lendStocksTransaction *Transaction, tx *gorm.DB) error {
	// subtracting available stocks before saving lending transaction
	if err := updateShortSellBank(lendStocksTransaction.StockId, -uint32(lendStocksTransaction.StockQuantity), tx); err != nil {
		return err
	}

	// saving transaction
	if err := tx.Save(lendStocksTransaction).Error; err != nil {
		return err
	}

	// saving lend
	if err := createShortSellLend(lendStocksTransaction.StockId, lendStocksTransaction.UserId, uint32(lendStocksTransaction.StockQuantity), tx); err != nil {
		return err
	}

	return nil
}

// update available stocks for shorting for a stock
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

// creates a short sell lend for short selling order
func createShortSellLend(stockId, userId, stockQuantity uint32, tx *gorm.DB) error {
	l := logger.WithFields(logrus.Fields{
		"method":  "createShortSellLend",
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

/*
squareOffLends square off the active intra day lends
- takes back the stocks given to the user(from stocks they owns) on that day
- the difference in the worth is the profit/loss the user made in shorting

**must be called after market is closed**
*/
func SquareOffLends() error {
	l := logger.WithFields(logrus.Fields{
		"method": "squareOffLends",
	})

	l.Debug("Attempting to square off active lends")

	db := getDB()

	var shortSellActiveLends []ShortSellLends

	if err := db.Find(&shortSellActiveLends).Error; err != nil {
		l.Errorf("error fetching active lends from db Error : %+v", err)
		return err
	}

	for _, lend := range shortSellActiveLends {
		tx := db.Begin() // begin Transaction

		allStocks.m[lend.StockId].RLock()
		currentPrice := allStocks.m[lend.StockId].stock.CurrentPrice
		allStocks.m[lend.StockId].RUnlock()

		shortSellTransaction := GetTransactionRef(lend.UserId, lend.StockId, ShortSellTransaction, 0, -int64(lend.StockQuantity), currentPrice, 0, 0)

		l.Infof("Saving ShortsellTransaction, stockId : %d, stockQuantity : %d, userId : %d", lend.StockId, lend.StockQuantity, lend.UserId)

		if err := tx.Save(shortSellTransaction).Error; err != nil {
			l.Errorf("rolling back, error saving shortsell transaction %+v", err)
			tx.Rollback()
			return err
		}

		lend.IsSquaredOff = true
		if err := tx.Save(lend).Error; err != nil {
			l.Errorf("rolling back, error updating shortSellLends %+v", err)
			tx.Rollback()
			return err
		}

		// restore the stock quantity back to shortsellbank
		if err := updateShortSellBank(lend.StockId, lend.StockQuantity, tx); err != nil {
			l.Errorf("rolling back, error updating shortsellbank %+v", err)
			tx.Rollback()
			return err
		}

		// commit transaction
		if err := tx.Commit().Error; err != nil {
			l.Errorf("error commiting the transaction %+v", err)
			return err
		}
	}

	l.Info("squared off all the active short sell lends")

	return nil
}

func getUserShortSellStocks(stockId, userId uint32) (uint32, error) {
	l := logger.WithFields(logrus.Fields{
		"method":  "getUserShortSellStocks",
		"stockId": stockId,
		"userId":  userId,
	})

	l.Debug("Attempting to fetch user current short sell stocks")

	sql := "SELECT SUM(stockQuantity) AS stockQty FROM ShortSellLends Where isSquaredOff = 0 AND stockId = ? AND userId = ?"

	db := getDB()

	stockQty := struct {
		StockQuantity uint32 `gorm:"column:stockQty"`
	}{
		StockQuantity: 0,
	}

	if err := db.Raw(sql, stockId, userId).Scan(&stockQty).Error; err != nil {
		l.Errorf("error fetching data from db %+v", err)
		return 0, err
	}

	return stockQty.StockQuantity, nil
}
