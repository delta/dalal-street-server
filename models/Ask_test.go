package models

import (
	"testing"

	"github.com/delta/dalal-street-server/utils/test"
	"github.com/sirupsen/logrus"
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

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
		t.Fatal("Converted value not equal")
	}
}

func Test_GetMyOpenAsks(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_GetMyOpenAsks",
	})

	var makeAsk = func(userId uint32, stockId uint32, isClosed bool) *Ask {
		return &Ask{
			UserId:   userId,
			StockId:  stockId,
			IsClosed: isClosed,
		}
	}

	user := &User{Id: 3}

	stocks := []*Stock{
		{Id: 1},
		{Id: 2},
	}

	asks := []*Ask{
		makeAsk(3, 1, false),
		makeAsk(3, 1, true),
		makeAsk(3, 1, false),
		makeAsk(3, 2, false),
		makeAsk(3, 2, false),
		makeAsk(3, 2, true),
	}

	db := getDB()

	defer func() {
		for _, ask := range asks {
			db.Delete(ask)
		}
		for _, stock := range stocks {
			db.Delete(stock)
		}
		db.Delete(user)
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

	openAsks, err := GetMyOpenAsks(3)

	if err != nil {
		l.Errorf("Errored in GetMyOpenAsks : %+v", err)
	}

	expectedReturnValue := []*Ask{
		asks[0],
		asks[2],
		asks[3],
		asks[4],
	}

	if !testutils.AssertEqual(t, openAsks, expectedReturnValue) {
		t.Fatalf("Got %+v; want %+v", openAsks, expectedReturnValue)
	}

}

func Test_GetMyClosedAsks(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_GetMyClosedAsks",
	})

	var makeAsk = func(userId uint32, stockId uint32, isClosed bool) *Ask {
		return &Ask{
			UserId:   userId,
			StockId:  stockId,
			IsClosed: isClosed,
		}
	}

	user := &User{Id: 3}

	stocks := []*Stock{
		{Id: 1},
		{Id: 2},
	}

	asks := []*Ask{
		makeAsk(3, 1, false),
		makeAsk(3, 1, true),
		makeAsk(3, 1, false),
		makeAsk(3, 2, false),
		makeAsk(3, 2, false),
		makeAsk(3, 2, true),
	}

	db := getDB()

	defer func() {
		for _, ask := range asks {
			db.Delete(ask)
		}
		for _, stock := range stocks {
			db.Delete(stock)
		}
		db.Delete(user)
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

	moreExists, closedAsks, err := GetMyClosedAsks(3, 0, 0)

	if err != nil {
		l.Errorf("Errored in GetMyClosedAsks : %+v", err)
	}

	expectedReturnValue := []*Ask{
		asks[5],
		asks[1],
	}

	if !testutils.AssertEqual(t, closedAsks, expectedReturnValue) {
		t.Fatalf("Got %+v; want %+v", closedAsks, expectedReturnValue)
	}

	if moreExists {
		t.Fatalf("moreExists returned true but it should be false")
	}

	moreExists, closedAsks, err = GetMyClosedAsks(3, 0, 1)

	if err != nil {
		l.Errorf("Errored in GetMyClosedAsks : %+v", err)
	}

	expectedReturnValue = []*Ask{
		asks[5],
	}

	if !testutils.AssertEqual(t, closedAsks, expectedReturnValue) {
		t.Fatalf("Got %+v; want %+v", closedAsks, expectedReturnValue)
	}

	if !moreExists {
		t.Fatalf("moreExists returned false but it should be true")
	}

}
