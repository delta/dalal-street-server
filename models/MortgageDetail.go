package models

import (
	"github.com/Sirupsen/logrus"
	//models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

// type MortgageDetail struct {
// 	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
// 	UserId       uint32 `gorm:"column:userId;not null" json:"user_id"`
// 	StockId      uint32 `gorm:"column:stockId;not null" json:"stock_id"`
// 	StocksInBank uint32 `gorm:"column:stocksInBank;not null" json:"stocks_in_bank"`
// }

// func (MortgageDetail) TableName() string {
// 	return "MortgageDetails"
// }

// func (md *MortgageDetail) ToProto() *models_proto.MortgageDetail {
// 	return &models_proto.MortgageDetail{
// 		Id:           md.Id,
// 		UserId:       md.UserId,
// 		StockId:      md.StockId,
// 		StocksInBank: md.StocksInBank,
// 	}
// }

type MortgageQueryData struct {
	StockId uint32
	StocksInBank int32
}

func GetMortgageDetails(userId uint32) ([]MortgageQueryData, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMortgageDetails",
		"userId": userId,
	})

	l.Infof("Attempting to get mortgageDetails for userId : %v", userId)

	db, err := DbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var mortgageDetails []MortgageQueryData

	db.Raw("SELECT stockId AS stock_id, SUM(stockQuantity) AS stocks_in_bank FROM Transactions WHERE userId = ? AND type = ? GROUP BY stockId", userId, "MortgageTransaction").Scan(&mortgageDetails)

	l.Infof("Successfully fetched mortgageDetails for userId : %v", userId)
	return mortgageDetails, nil
}
