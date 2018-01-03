package models

import (
	"testing"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestStockToProto(t *testing.T) {
	o := &Stock{
		Id:               23,
		ShortName:        "zold",
		FullName:         "PastCry",
		Description:      "This Stock is a stock :P",
		CurrentPrice:     200,
		DayHigh:          300,
		DayLow:           100,
		AllTimeHigh:      400,
		AllTimeLow:       90,
		StocksInExchange: 123,
		StocksInMarket:   234,
		UpOrDown:         true,
		PreviousDayClose: 1000,
		AvgLastPrice:     120,
		CreatedAt:        "2017-02-09T00:00:00",
		UpdatedAt:        "2017-02-09T00:00:00",
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}

func Test_UpdateStockPrice(t *testing.T) {
	stock := &Stock{
		Id:               1,
		CurrentPrice:     1000,
		AllTimeLow:       1000,
		PreviousDayClose: 1000,
		AllTimeHigh:      1000,
	}

	db, err := DbOpen()
	if err != nil {
		t.Fatalf("Opening Database for inserting stocks failed  %v", err)
	}
	defer db.Close()

	db.Save(stock)
	defer func() {
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
	}()

	LoadStocks()
	err = UpdateStockPrice(1, 2000)
	if err != nil {
		t.Fatalf("UpdateStockPrice failed with %+v", err)
	}

	retrievedStock := &Stock{}
	db.First(retrievedStock, 1)
	var stock1 = &Stock{
		Id:               1,
		CurrentPrice:     2000,
		DayHigh:          2000,
		AllTimeHigh:      2000,
		AllTimeLow:       1000,
		UpOrDown:         true,
		AvgLastPrice:     850,
		PreviousDayClose: 1000,
	}
	if !testutils.AssertEqual(t, stock1, retrievedStock) {
		t.Fatalf("Expected %v but got %v", stock1, retrievedStock)
	}
}
func Test_GetCompanyDetails(t *testing.T) {
	var stock = &Stock{Id: 1, CurrentPrice: 1000}
	db, err := DbOpen()
	if err != nil {
		t.Fatalf("Opening data base for inserting stocks failed %v", err)
	}
	defer db.Close()
	db.Save(stock)
	defer func() {
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
	}()
	LoadStocks()
	testutils.Sleep(90)
	_, retrievedStockList, _ := GetCompanyDetails(1)
	var stkHistoryPoint = &StockHistory{
		StockId:    1,
		StockPrice: 1000,
		CreatedAt:  retrievedStockList[0].CreatedAt,
	}
	if !testutils.AssertEqual(t, stkHistoryPoint, retrievedStockList[0]) {
		t.Fatalf("Expected %v but got %v", stkHistoryPoint, retrievedStockList[0])
	}
}
func Test_AddStocksToExchange(t *testing.T) {
	var stock = &Stock{Id: 1, CurrentPrice: 1000, StocksInMarket: 123, StocksInExchange: 234}
	db, err := DbOpen()
	if err != nil {
		t.Fatalf("Opening data base for inserting stocks failed %v", err)
	}
	defer db.Close()
	db.Save(stock)
	defer func() {
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
	}()
	LoadStocks()
	AddStocksToExchange(1, 10)
	var retrievedStock = &Stock{Id: 1}
	db.First(retrievedStock)
	var stockEqual = &Stock{Id: 1, CurrentPrice: 1000, StocksInMarket: 123, StocksInExchange: 244}
	if !testutils.AssertEqual(t, retrievedStock, stockEqual) {
		t.Fatalf("Expected %v but got %v", stockEqual, retrievedStock)
	}
}
