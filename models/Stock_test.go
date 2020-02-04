package models

import (
	"testing"

	"github.com/delta/dalal-street-server/utils"
	testutils "github.com/delta/dalal-street-server/utils/test"
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
		LastTradePrice:   120,
		RealAvgPrice:     200,
		CreatedAt:        "2017-02-09T00:00:00",
		UpdatedAt:        "2017-02-09T00:00:00",
		GivesDividends:   true,
	}

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
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
		StocksInExchange: 0,
		StocksInMarket:   200,
		RealAvgPrice:     1000,
	}

	db := getDB()

	db.Save(stock)
	defer func() {
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
	}()

	LoadStocks()
	err := UpdateStockPrice(1, 2000, 1)
	if err != nil {
		t.Fatalf("UpdateStockPrice failed with %+v", err)
	}

	retrievedStock := &Stock{}
	db.First(retrievedStock, 1)
	var stock1 = &Stock{
		Id:               1,
		CurrentPrice:     1500,
		DayHigh:          1500,
		AllTimeHigh:      1500,
		AllTimeLow:       1000,
		UpOrDown:         true,
		LastTradePrice:   2000,
		PreviousDayClose: 1000,
		StocksInExchange: 0,
		StocksInMarket:   200,
		RealAvgPrice:     1500,
		UpdatedAt:        retrievedStock.UpdatedAt,
	}
	if !testutils.AssertEqual(t, stock1, retrievedStock) {
		t.Fatalf("Expected %v but got %v", stock1, retrievedStock)
	}
}

func Test_GetCompanyDetails(t *testing.T) {
	var stock = &Stock{
		Id:               1,
		ShortName:        "XRP",
		FullName:         "Ripple XRP",
		Description:      "Ripple description",
		CurrentPrice:     1000,
		DayHigh:          1500,
		DayLow:           1400,
		AllTimeHigh:      2000,
		AllTimeLow:       1000,
		StocksInExchange: 150,
		StocksInMarket:   140,
		PreviousDayClose: 1300,
		UpOrDown:         true,
		CreatedAt:        utils.GetCurrentTimeISO8601(),
		UpdatedAt:        utils.GetCurrentTimeISO8601(),
	}

	db := getDB()

	db.Save(stock)
	defer func() {
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
	}()

	LoadStocks()
	retrievedStock, err := GetCompanyDetails(1)

	if err != nil {
		t.Fatalf("Unexpected error %+v", err)
	}

	if !testutils.AssertEqual(t, retrievedStock, stock) {
		t.Fatalf("Expected %v but got %v", retrievedStock, stock)
	}
}

func Test_AddStocksToExchange(t *testing.T) {
	var stock = &Stock{Id: 1, CurrentPrice: 1000, StocksInMarket: 123, StocksInExchange: 234}
	db := getDB()

	db.Save(stock)
	defer func() {
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
	}()

	LoadStocks()
	AddStocksToExchange(1, 10)

	var retrievedStock = &Stock{Id: 1}
	db.First(retrievedStock)
	var stockEqual = &Stock{Id: 1, CurrentPrice: 1000, StocksInMarket: 123, StocksInExchange: 244, UpdatedAt: retrievedStock.UpdatedAt}

	if !testutils.AssertEqual(t, retrievedStock, stockEqual) {
		t.Fatalf("Expected %v but got %v", stockEqual, retrievedStock)
	}
}
