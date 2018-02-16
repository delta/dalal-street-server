package models

import (
	"github.com/Sirupsen/logrus"
)

// MortgageQueryData stores stocks in bank of a given stockid
type MortgageQueryData struct {
	StockId      uint32
	StocksInBank int32
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

	db.Raw("SELECT stockId AS stock_id, SUM(stockQuantity) AS stocks_in_bank FROM Transactions WHERE userId = ? AND type = ? GROUP BY stockId", userID, "MortgageTransaction").Scan(&mortgageDetails)

	l.Infof("Successfully fetched mortgageDetails for userID : %v", userID)
	return mortgageDetails, nil
}
