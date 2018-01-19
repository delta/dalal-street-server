package models

import (
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// func TestMortgageDetailToProto(t *testing.T) {
// 	o := &MortgageDetail{
// 		Id:           1,
// 		UserId:       20,
// 		StockId:      300,
// 		StocksInBank: 10,
// 	}

// 	o_proto := o.ToProto()

// 	if !testutils.AssertEqual(t, o, o_proto) {
// 		t.Fatal("Converted values not equal!")
// 	}
// }

// TODO: fix this test!
func Test_GetMortgageDetails(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_GetMortgageDetails",
	})

	var makeTrans = func(userId uint32, stockId uint32, transType TransactionType, stockQty int32, price uint32, total int32) *Transaction {
		return &Transaction{
			UserId:        userId,
			StockId:       stockId,
			Type:          transType,
			StockQuantity: stockQty,
			Price:         price,
			Total:         total,
		}
	}

	var makeUser = func(id uint32, email string, name string, cash uint32, total int32) *User {
		return &User{
			Id:        id,
			Email:     email,
			Name:      name,
			Cash:      cash,
			Total:     total,
			CreatedAt: utils.GetCurrentTimeISO8601(),
		}
	}

	var makeStock = func(id uint32, sName string, fName string, desc string, curPrice uint32, dayHigh uint32, dayLow uint32, allHigh uint32, allLow uint32, stocks uint32, upOrDown bool) *Stock {
		return &Stock{
			Id:               id,
			ShortName:        sName,
			FullName:         fName,
			Description:      desc,
			CurrentPrice:     curPrice,
			DayHigh:          dayHigh,
			DayLow:           dayLow,
			AllTimeHigh:      allHigh,
			AllTimeLow:       allLow,
			StocksInExchange: stocks,
			UpOrDown:         upOrDown,
			CreatedAt:        utils.GetCurrentTimeISO8601(),
			UpdatedAt:        utils.GetCurrentTimeISO8601(),
		}
	}

	users := []*User{
		makeUser(2, "test@testmail.com", "Test", 100000, 100000),
	}

	stocks := []*Stock{
		makeStock(1, "FB", "Facebook", "Social", 100, 200, 60, 300, 10, 2000, true),
		makeStock(2, "MS", "Microsoft", "MS Corp", 300, 450, 60, 600, 10, 2000, true),
	}

	transactions := []*Transaction{
		makeTrans(2, 1, MortgageTransaction, -10, 200, 2000),
		makeTrans(2, 1, MortgageTransaction, -40, 200, 8000),
		makeTrans(2, 1, MortgageTransaction, 10, 200, -2000),
		makeTrans(2, 1, MortgageTransaction, -40, 200, 8000),
		makeTrans(2, 2, MortgageTransaction, -15, 300, 4500),
		makeTrans(2, 2, MortgageTransaction, -30, 300, 9000),
		makeTrans(2, 2, MortgageTransaction, -15, 300, 4500),
		makeTrans(2, 2, MortgageTransaction, 30, 300, -9000),
	}

	db, err := DbOpen()
	if err != nil {
		t.Fatal("Failed opening DB to insert dummy data")
	}

	defer func() {
		for _, tr := range transactions {
			db.Delete(tr)
		}
		for _, stock := range stocks {
			db.Delete(stock)
		}
		for _, user := range users {
			db.Delete(user)
		}
		db.Close()
	}()

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}

	for _, stock := range stocks {
		if err := db.Create(stock).Error; err != nil {
			t.Fatal(err)
		}
	}

	for _, tr := range transactions {
		if err := db.Create(tr).Error; err != nil {
			t.Fatal(err)
		}
	}

	mortgageDetailsTestResponse, err := GetMortgageDetails(2)

	l.Infof("Response : %+v", mortgageDetailsTestResponse)
}
