package models

import (
	"github.com/thakkarparth007/dalal-street-server/utils/test"
	"github.com/Sirupsen/logrus"
	"time"
	"testing"
)

func TestAskToProto(t *testing.T) {
	o := &Ask{
		Id:                     2,
		UserId:                 2,
		StockId:                3,
		Price:                  5,
		OrderType:              Market,
		StockQuantity:          20,
		StockQuantityFulfilled: 20,
		IsClosed:               true,
		CreatedAt:              "2017-02-09T00:00:00",
		UpdatedAt:              "2017-02-09T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}

func Test_GetMyAsks(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "Test_GetMyAsks",
	})

	l.Infof("Attempting to get asks for user id %d", 3)

	var makeUser = func(id uint32, email string, name string, cash uint32, total int32) *User {
		return &User{
			Id:        id,
			Email:     email,
			Name:      name,
			Cash:      cash,
			Total:     total,
			CreatedAt: time.Now().Format(time.RFC3339),
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
			CreatedAt:        time.Now().Format(time.RFC3339),
			UpdatedAt:        time.Now().Format(time.RFC3339),
		}
	}

	var makeAsk = func(userId uint32, stockId uint32, ot OrderType, stockQty uint32, price uint32, isClosed bool) *Ask {
		return &Ask{
			UserId:        userId,
			StockId:       stockId,
			OrderType:     ot,
			StockQuantity: stockQty,
			Price:         price,
			IsClosed:      isClosed,
		}
	}

	user := makeUser(3, "lol@lol.com", "LOL", 99999, 99999)

	stocks := []*Stock{
		makeStock(1, "FB", "Facebook", "Social", 100, 200, 60, 300, 10, 2000, true),
		makeStock(2, "MS", "Microsoft", "MS Corp", 300, 450, 60, 600, 10, 2000, true),
	}

	asks := []*Ask{
		makeAsk(3, 1, Limit, 10, 100, false),
		makeAsk(3, 1, Limit, 20, 200, true),
		makeAsk(3, 1, Limit, 30, 100, false),
		makeAsk(3, 2, Limit, 50, 10, false),
		makeAsk(3, 2, Limit, 5, 25, false),
		makeAsk(3, 2, Limit, 75, 2, true),
	}

	db, err := DbOpen()
	if err != nil {
		t.Fatal("Failed opening DB to insert dummy data")
	}

	defer func() {
		for _, ask := range asks {
			db.Delete(ask)
		}
		for _, stock := range stocks {
			db.Delete(stock)
		}
		db.Delete(user)
		db.Close()
	}()

	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	for _, stock := range stocks {
		if err := db.Create(stock).Error; err != nil {
			t.Fatal(err)
		}
	}

	for _, ask := range asks {
		if err := db.Create(ask).Error; err != nil {
			t.Fatal(err)
		}
	}

	moreExists, openAsks, closedAsks, err := GetMyAsks(3,0,0)

	if err != nil {
		l.Errorf("Errored in GetMyAsks : %+v", err)
	}

	l.Infof("Received from GetMyAsks : %v %+v %+v", moreExists, openAsks, closedAsks)
}
