package models

import (
	"fmt"
	"testing"

	testutils "github.com/delta/dalal-street-server/utils/test"
)

func TestIpoStockToProto(t *testing.T) {
	o := &IpoStock{
		Id:            23,
		ShortName:     "BRKA",
		FullName:      "Berkshire Hathaway Inc",
		Description:   "Google me and find out :)",
		CreatedAt:     "2022-02-09T00:00:00",
		UpdatedAt:     "2022-02-09T00:00:00",
		IsBiddable:    true,
		SlotPrice:     10,
		StockPrice:    10,
		SlotQuantity:  10,
		StocksPerSlot: 10,
	}

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
		t.Fatal("Converted value not equal")
	}
}

// func TestGetAllStocks(t *testing.T) {
// 	stock := &Stock{
// 		// Id:               10010,
// 		CurrentPrice:     1000,
// 		AllTimeLow:       1000,
// 		PreviousDayClose: 1000,
// 		AllTimeHigh:      1000,
// 		StocksInExchange: 0,
// 		StocksInMarket:   200,
// 		RealAvgPrice:     1000,
// 	}
// 	db := getDB()
// 	if err := db.Create(stock).Error; err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Print(stock.Id)

// 	var TestallStocksMap = make(map[uint32]*Stock)
// 	TestallStocksMap = GetAllStocks()
// 	fmt.Printf("lenght of map : %+v", len(TestallStocksMap))
// 	fmt.Printf("all stocks in test : %+v", TestallStocksMap)
// }

// func WWWTest_GetAllIpoStocks(t *testing.T) {
// 	ipoStock := &IpoStock{StocksPerSlot: 20, IsBiddable: true}
// 	db := getDB()
// 	if err := db.Create(ipoStock).Error; err != nil {
// 		t.Fatal(err)
// 	}
// 	var TestallIpoStocksMap = make(map[uint32]*IpoStock)
// 	TestallIpoStocksMap = GetAllIpoStocks()
// 	fmt.Printf("lenght of map : %+v", len(TestallIpoStocksMap))
// 	fmt.Printf("all ipo stocks in test : %+v", TestallIpoStocksMap)
// }

func Test_AllowIpoBidding(t *testing.T) {

	IpoStock1 := &IpoStock{
		ShortName:     "BRKA",
		FullName:      "Berkshire Hathaway Inc",
		Description:   "Google me and find out :)",
		CreatedAt:     "2022-02-09T00:00:00",
		UpdatedAt:     "2022-02-09T00:00:00",
		IsBiddable:    false,
		SlotPrice:     1000,
		StockPrice:    100,
		SlotQuantity:  10,
		StocksPerSlot: 10,
	}
	users := []*User{
		{Id: 201, Cash: 100000, Email: "201@gmail.com"},
		{Id: 202, Cash: 100000, Email: "202@gmail.com"},
	}

	db := getDB()
	defer func() {
		db.Exec("DELETE FROM IpoBids")
		db.Exec("DELETE FROM IpoStocks")
		for _, user := range users {
			db.Delete(user)
		}
	}()

	if err := db.Create(IpoStock1).Error; err != nil {
		t.Fatal(err)
	}

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}

	ipoBidId1, err := CreateIpoBid(201, IpoStock1.Id, 1, 10000) // fails because ipo stock is not biddable
	if err != nil {
		fmt.Printf("Expected Error in CreateIpoBid1 : %+v \n", err)
	}

	if err := AllowIpoBidding(IpoStock1.Id); err != nil {
		t.Fatalf("\n  %d", err)
	}

	ipoBidId2, err := CreateIpoBid(202, IpoStock1.Id, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid2 : %+v", err)
	}

	if err := AllowIpoBidding(IpoStock1.Id); err != nil {
		fmt.Printf("Expected Error in AllowIpoBidding  %d \n", err)
	} // error because IPO stock already opened for bidding

	fmt.Printf("IpoBidId1 : %d\n IpoBidId2: %d\n", ipoBidId1, ipoBidId2)
}

func Test_AllotIpoSlots(t *testing.T) {
	users := []*User{
		{Id: 301, Cash: 100000, Email: "301@gmail.com"},
		{Id: 302, Cash: 100000, Email: "302@gmail.com"},
		{Id: 303, Cash: 100000, Email: "303@gmail.com"},
		{Id: 304, Cash: 100000, Email: "304@gmail.com"},
		{Id: 305, Cash: 100000, Email: "305@gmail.com"},
		{Id: 306, Cash: 100000, Email: "306@gmail.com"},
	}

	ipoStock1 := &IpoStock{
		ShortName:     "ABC",
		FullName:      "TestStock1",
		Description:   "ABC",
		CreatedAt:     "2022-02-09T00:00:00",
		UpdatedAt:     "2022-02-09T00:00:00",
		IsBiddable:    true,
		SlotPrice:     10000,
		StockPrice:    500,
		SlotQuantity:  5,
		StocksPerSlot: 20,
	}
	ipoStock2 := &IpoStock{
		ShortName:     "ABC",
		FullName:      "TestStock2",
		Description:   "ABC",
		CreatedAt:     "2022-02-09T00:00:00",
		UpdatedAt:     "2022-02-09T00:00:00",
		IsBiddable:    true,
		SlotPrice:     10000,
		StockPrice:    500,
		SlotQuantity:  7,
		StocksPerSlot: 20,
	}

	db := getDB()

	defer func() {
		db.Exec("DELETE FROM IpoBids")
		db.Exec("DELETE FROM IpoStocks")
		db.Exec("DELETE FROM Transactions")
		db.Delete(ipoStock1)
		db.Delete(ipoStock2)
		for _, user := range users {
			db.Delete(user)
		}
	}()

	if err := db.Create(ipoStock1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(ipoStock2).Error; err != nil {
		t.Fatal(err)
	}

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}

	ipoStockid := ipoStock1.Id

	fmt.Printf("\n\n Over Subscription: \n\n")
	ipoBidId1, err := CreateIpoBid(301, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId2, err := CreateIpoBid(302, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId3, err := CreateIpoBid(303, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId4, err := CreateIpoBid(304, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId5, err := CreateIpoBid(305, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId6, err := CreateIpoBid(306, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	fmt.Printf("IpoBids ids = %d  %d  %d  %d  %d  %d \n", ipoBidId1, ipoBidId2, ipoBidId3, ipoBidId4, ipoBidId5, ipoBidId6)

	if err := AllotSlots(ipoStockid); err != nil {
		t.Fatal(err)
	}
	var FulfilledIpoBids []*IpoBid
	if err := db.Where("ipoStockId = ? AND isFulfilled = ?", ipoStockid, true).Find(&FulfilledIpoBids).Error; err != nil {
		t.Fatal(err)
	}

	fmt.Printf("FulfilledIpoBids  Ids: ")
	for _, FulfilledIpoBid := range FulfilledIpoBids {
		fmt.Printf("  %+v  ", FulfilledIpoBid.Id)
	}

	NewStock := &Stock{}
	if err := db.Where("fullName = ?", "TestStock1").First(&NewStock).Error; err != nil {
		fmt.Print(err)
	}
	fmt.Printf("\n listingprice = %+v  StocksInMarket = %+v \n\n", NewStock.CurrentPrice, NewStock.StocksInMarket)

	var newusers []*User

	if err := db.Find(&newusers).Error; err != nil {
		t.Fatal(err)
	}

	for _, newuser := range newusers {
		fmt.Printf("UserId= %+v  cash = %+v  ReservedCash= %+v  \n", newuser.Id, newuser.Cash, newuser.ReservedCash)
	}
	// Expected Output:
	// listingprice = 550   StocksInMarket = 100
	// all users with cash 90000, reservedCash 10000

	fmt.Printf("\n\n Under Subscription: \n\n")

	ipoStockid = ipoStock2.Id

	ipoBidId1, err = CreateIpoBid(301, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId2, err = CreateIpoBid(302, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId3, err = CreateIpoBid(303, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId4, err = CreateIpoBid(304, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId5, err = CreateIpoBid(305, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	ipoBidId6, err = CreateIpoBid(306, ipoStockid, 1, 10000)
	if err != nil {
		t.Fatalf("Errored in CreateIpoBid : %+v", err)
	}
	fmt.Printf("IpoBids ids = %d  %d  %d  %d  %d  %d \n", ipoBidId1, ipoBidId2, ipoBidId3, ipoBidId4, ipoBidId5, ipoBidId6)

	if err := AllotSlots(ipoStockid); err != nil {
		t.Fatal(err)
	}

	NewStock1 := &Stock{}
	if err := db.Where("fullName = ?", "TestStock2").First(&NewStock1).Error; err != nil {
		fmt.Print(err)
	}
	fmt.Printf("\n listingprice = %+v   StocksInMarket = %+v\n\n", NewStock1.CurrentPrice, NewStock1.StocksInMarket)

	if err := db.Find(&newusers).Error; err != nil {
		t.Fatal(err)
	}

	for _, newuser := range newusers {
		fmt.Printf("UserId= %+v  cash = %+v   ReservedCash= %+v \n", newuser.Id, newuser.Cash, newuser.ReservedCash)
	}
	// Expected Output:
	// listingprice = 475   StocksInMarket = 120
	// 5 users with cash 90000, 1 user with cash 90000, all users with reservedCash 0

}
