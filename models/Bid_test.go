package models

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils/test"
)

func TestBidToProto(t *testing.T) {
	o := &Bid{
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

func Test_GetMyOpenBids(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_GetMyOpenBids",
	})

	var makeBid = func(userId uint32, stockId uint32, isClosed bool) *Bid {
		return &Bid{
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

	bids := []*Bid{
		makeBid(3, 1, false),
		makeBid(3, 1, true),
		makeBid(3, 1, false),
		makeBid(3, 2, false),
		makeBid(3, 2, false),
		makeBid(3, 2, true),
	}

	db := getDB()

	defer func() {
		for _, bid := range bids {
			db.Delete(bid)
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

	for _, bid := range bids {
		if err := db.Create(bid).Error; err != nil {
			t.Fatal(err)
		}
	}

	openBids, err := GetMyOpenBids(3)

	if err != nil {
		l.Errorf("Errored in GetMyOpenBids : %+v", err)
	}

	expectedReturnValue := []*Bid{
		bids[0],
		bids[2],
		bids[3],
		bids[4],
	}

	if !testutils.AssertEqual(t, openBids, expectedReturnValue) {
		t.Fatalf("Got %+v; want %+v", openBids, expectedReturnValue)
	}

}

func Test_GetMyClosedBids(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_GetMyClosedBids",
	})

	var makeBid = func(userId uint32, stockId uint32, isClosed bool) *Bid {
		return &Bid{
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

	bids := []*Bid{
		makeBid(3, 1, false),
		makeBid(3, 1, true),
		makeBid(3, 1, false),
		makeBid(3, 2, false),
		makeBid(3, 2, false),
		makeBid(3, 2, true),
	}

	db := getDB()

	defer func() {
		for _, bid := range bids {
			db.Delete(bid)
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

	for _, bid := range bids {
		if err := db.Create(bid).Error; err != nil {
			t.Fatal(err)
		}
	}

	moreExists, closedBids, err := GetMyClosedBids(3, 0, 0)

	if err != nil {
		l.Errorf("Errored in GetMyClosedBids : %+v", err)
	}

	expectedReturnValue := []*Bid{
		bids[5],
		bids[1],
	}

	if !testutils.AssertEqual(t, closedBids, expectedReturnValue) {
		t.Fatalf("Got %+v; want %+v", closedBids, expectedReturnValue)
	}

	if moreExists {
		t.Fatalf("moreExists returned true but it should be false")
	}

	moreExists, closedBids, err = GetMyClosedBids(3, 0, 1)

	if err != nil {
		l.Errorf("Errored in GetMyClosedBids : %+v", err)
	}

	expectedReturnValue = []*Bid{
		bids[5],
	}

	if !testutils.AssertEqual(t, closedBids, expectedReturnValue) {
		t.Fatalf("Got %+v; want %+v", closedBids, expectedReturnValue)
	}

	if !moreExists {
		t.Fatalf("moreExists returned false but it should be true")
	}

}
