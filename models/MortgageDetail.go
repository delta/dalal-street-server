package models

import (
	"github.com/Sirupsen/logrus"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/jinzhu/gorm"
)

// MortgageQueryData stores stocks in bank of a given stockid
type MortgageQueryData struct {
	StockID       uint32
	StocksInBank  uint64
	MortgagePrice uint64
}

// GetMortgageDetails returns mortgage data about a user
func GetMortgageDetails(userID uint32) ([]MortgageQueryData, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMortgageDetails",
		"userID": userID,
	})

	l.Infof("Attempting to get mortgageDetails for userID : %v", userID)

	db := getDB()

	var mortgageDetails []MortgageQueryData

	db.Raw("SELECT stockId AS stock_id, stocksInBank AS stocks_in_bank, mortgagePrice AS mortgage_price  FROM MortgageDetails WHERE userId = ?", userID).Scan(&mortgageDetails)

	l.Infof("Successfully fetched mortgageDetails for userID : %v", userID)
	return mortgageDetails, nil
}

func (m *MortgageQueryData) ToProto() *models_pb.MortgageDetail {
	return &models_pb.MortgageDetail{
		StockId:       m.StockID,
		StocksInBank:  m.StocksInBank,
		MortgagePrice: m.MortgagePrice,
	}
}

// mortgageStocksAction returns total transaction amount while mortgaging
func mortgageStocksAction(user *User, stockID uint32, stockQuantity int64, mortgagePrice uint64, tx *gorm.DB) (int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "mortgageStocksAction",
		"param_userId":        user.Id,
		"param_stockId":       stockID,
		"param_stockQuantity": stockQuantity,
		"param_mortgagePrice": mortgagePrice,
	})

	stockOwned, err := getSingleStockCount(user, stockID)
	if err != nil {
		l.Error(err)
		return 0, err
	}

	if stockQuantity > stockOwned {
		l.Errorf("Insufficient stocks to mortgage. Have %d, want %d", stockOwned, stockQuantity)
		return 0, NotEnoughStocksError{}
	}

	stocksInBank, err := getStocksInBank(user.Id, stockID, mortgagePrice, tx)
	if (err != nil && err != InvalidRetrievePriceError{}) /* This type of error not applicable here */ {
		l.Error(err)
		return 0, err
	}

	if stocksInBank == 0 {
		sql := "INSERT into MortgageDetails (userId, stockId, stocksInBank, mortgagePrice) VALUES (?, ?, ?, ?)"
		err = tx.Exec(sql, user.Id, stockID, stockQuantity, mortgagePrice).Error
	} else {
		sql := "UPDATE MortgageDetails SET stocksInBank=? WHERE userId=? AND stockId=? AND mortgagePrice=?"
		err = tx.Exec(sql, stocksInBank+stockQuantity, user.Id, stockID, mortgagePrice).Error
	}

	if err != nil {
		l.Error(err)
		return 0, err
	}

	return int64(mortgagePrice) * stockQuantity * MORTGAGE_DEPOSIT_RATE / 100, nil
}

// retrieveStocksAction returns total transaction amount and stock quantity which could be retrieved
func retrieveStocksAction(userID, stockID uint32, stockQuantity int64, userCash, retrievePrice uint64, tx *gorm.DB) (int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "retrieveStocksAction",
		"param_userId":        userID,
		"param_stockId":       stockID,
		"param_stockQuantity": stockQuantity,
		"param_retrievePrice": retrievePrice,
	})

	l.Debugf("Retrieving stocks in action")

	stocksInBank, err := getStocksInBank(userID, stockID, retrievePrice, tx)
	if err != nil {
		l.Error(err)
		return 0, err
	}

	if int64(userCash/retrievePrice) < stockQuantity {
		l.Errorf("Insufficient cash with user. Have %d, want %d", userCash, stockQuantity*int64(retrievePrice))
		return 0, NotEnoughCashError{}
	}

	if stockQuantity < stocksInBank {
		sql := "UPDATE MortgageDetails SET stocksInBank=? WHERE userId=? AND stockId=? AND mortgagePrice=?"
		err = tx.Exec(sql, stocksInBank-stockQuantity, userID, stockID, retrievePrice).Error

	} else if stockQuantity == stocksInBank /* So we can delete that entire row */ {
		sql := "DELETE from MortgageDetails WHERE userId=? AND stockId=? AND mortgagePrice=?"
		err = tx.Exec(sql, userID, stockID, retrievePrice).Error

	} else /* stockQuantity to retrieve > stocksInBank */ {
		l.Errorf("Insufficient stocks in mortgage. Have %d, want %d", stocksInBank, stockQuantity)
		return 0, NotEnoughStocksError{}
	}

	if err != nil {
		l.Error(err)
		return 0, err
	}

	return -int64(retrievePrice) * stockQuantity * MORTGAGE_RETRIEVE_RATE / 100, nil
}

func getStocksInBank(userID, stockID uint32, retrievePrice uint64, tx *gorm.DB) (int64, error) {

	sql := "SELECT stocksInBank from MortgageDetails where userId=? AND stockId=? AND mortgagePrice=?"
	rows, err := tx.Raw(sql, userID, stockID, retrievePrice).Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, InvalidRetrievePriceError{}
	}

	var stocksInBank int64
	rows.Scan(&stocksInBank)

	return stocksInBank, nil
}
