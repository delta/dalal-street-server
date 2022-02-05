package models

import "testing"

func Test_GetAvailableLendStocks(t *testing.T) {
	stock := &Stock{Id: 1, CurrentPrice: 1000, StocksInMarket: 123, StocksInExchange: 234}
	db := getDB()
	db.Save(stock)

	defer func() {
		db.Exec("DELETE FROM ShortSellBank")
		db.Delete(stock)
	}()

	shortSellBank := &ShortSellBank{
		StockId:         stock.Id,
		AvailableStocks: 100,
	}

	db.Save(shortSellBank)

	availableStocks, err := getAvailableLendStocks(stock.Id)

	if err != nil {
		t.Error(err)
	}

	if availableStocks != shortSellBank.AvailableStocks {
		t.Fatalf("expected %d got %d", shortSellBank.AvailableStocks, availableStocks)
	}
}
