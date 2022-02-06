package models

import (
	"testing"

	testutils "github.com/delta/dalal-street-server/utils/test"
)

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
		t.Fatal(err)
	}

	if availableStocks != shortSellBank.AvailableStocks {
		t.Fatalf("expected %d got %d", shortSellBank.AvailableStocks, availableStocks)
	}
}

func Test_SaveLendStockTransaction(t *testing.T) {
	stock := &Stock{Id: 1, CurrentPrice: 1000, StocksInMarket: 123, StocksInExchange: 234}
	ssb := &ShortSellBank{StockId: 1, AvailableStocks: 10}
	user := &User{Id: 1}

	db := getDB()

	defer func() {
		db.Exec("DELETE FROM Transactions")
		db.Exec("DELETE FROM ShortSellLends")
		db.Exec("DELETE FROM ShortSellBank")
		db.Delete(stock)
		db.Delete(user)
	}()

	if err := db.Create(stock).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(ssb).Error; err != nil {
		t.Fatal(err)
	}

	transaction := GetTransactionRef(1, 1, ShortSellTransaction, 0, 5, 1000, 0, 5000)

	if err := saveLendStockTransaction(transaction, db); err != nil {
		t.Fatalf("error saving lend stock transaction %+v", err)
	}

	availableStocks, err := getAvailableLendStocks(stock.Id)

	if err != nil {
		t.Fatal(err)
	}

	if availableStocks != ssb.AvailableStocks-5 {
		t.Fatalf("Error, expected %v got %v", ssb.AvailableStocks-5, availableStocks)
	}

	expectedSsl := &ShortSellLends{
		StockId:       1,
		UserId:        1,
		StockQuantity: 5,
		IsSquaredOff:  false,
	}

	var savedSsl ShortSellLends

	if err := db.First(&savedSsl).Error; err != nil {
		t.Fatal(err)
	}

	if !testutils.AssertEqual(t, expectedSsl, savedSsl) {
		t.Fatalf("Expected %+v but got %+v", expectedSsl, savedSsl)
	}
}
