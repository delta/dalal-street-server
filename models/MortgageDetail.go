package models

import (
	"github.com/Sirupsen/logrus"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
)

// MortgageQueryData stores stocks in bank of a given stockid
type MortgageQueryData struct {
	StockId       uint32
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
		StockId:       m.StockId,
		StocksInBank:  m.StocksInBank,
		MortgagePrice: m.MortgagePrice,
	}
}
