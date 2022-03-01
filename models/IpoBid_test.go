package models

import (
	"fmt"
	"testing"

	testutils "github.com/delta/dalal-street-server/utils/test"
	"github.com/sirupsen/logrus"
)

func TestIpoBidToProto(t *testing.T) {
	o := &IpoBid{
		Id:           2,
		UserId:       2,
		IpoStockId:   1,
		SlotPrice:    5000,
		SlotQuantity: 100,
		IsFulfilled:  true,
		IsClosed:     true,
		CreatedAt:    "2022-02-09T00:00:00",
		UpdatedAt:    "2022-02-09T00:00:00",
	}

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
		t.Fatal("Converted value not equal")
	}
}

func Test_CreateIpoBid(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_CreateIpoBid",
	})

	users := []*User{
		{Id: 101, Cash: 100000, Email: "101@gmail.com"},
		{Id: 102, Cash: 100000, Email: "102@gmail.com"},
		{Id: 103, Cash: 100000, Email: "103@gmail.com"},
		{Id: 104, Cash: 100000, Email: "104@gmail.com"},
	}

	ipoStock := &IpoStock{Id: 1, StocksPerSlot: 20}

	db := getDB()

	defer func() {
		db.Exec("DELETE FROM IpoBids")
		db.Exec("DELETE FROM IpoStocks")
		for _, user := range users {
			db.Delete(user)
		}
		db.Delete(ipoStock)
	}()

	if err := db.Create(ipoStock).Error; err != nil {
		t.Fatal(err)
	}

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}

	ipoBidId1, err := CreateIpoBid(101, 1, 1, 10000)
	ipoBidId2, err := CreateIpoBid(102, 1, 1, 10000)
	ipoBidId3, err := CreateIpoBid(103, 1, 1, 10000)
	ipoBidId4, err1 := CreateIpoBid(104, 1, 1, 1000000000) // fails because of not enough cash
	ipoBidId5, err2 := CreateIpoBid(101, 1, 1, 10000)      // fails because of >1 bid (and thus >1 slotquantity) for same user
	ipoBidId6, err3 := CreateIpoBid(102, 1, 2, 10000)      // fails because of >1 slotquantity

	if err != nil {
		l.Errorf("Errored in CreateIpoBid : %+v", err)
	}

	fmt.Printf("IpoBids ids = %d  %d  %d  %d  %d  %d \n", ipoBidId1, ipoBidId2, ipoBidId3, ipoBidId4, ipoBidId5, ipoBidId6)
	fmt.Print("err1 = ", err1, " err2 = ", err2, " err3 = ", err3, "\n")
	// Expected Output : err1 = Not enough cash to place this order
	// err2 = An order can involve a trade of min 1 and max 50 stocks
	// err3 = An order can involve a trade of min 1 and max 50 stocks

	err = CancelIpoBid(ipoBidId3)
	err = CancelIpoBid(ipoBidId3) // fails because bid is already cancelled
	fmt.Println("CancelIpoBid err = ", err)
	// Expected Output : CancelIpoBid err =  Order#3 is already closed. Cannot cancel now.

	ipoBidId7, err := CreateIpoBid(103, 1, 1, 92000) // succeeds because bid is cancelled and has enough cash
	fmt.Println("ipoBidId7 = ", ipoBidId7)
	fmt.Println("err = ", err)

}
