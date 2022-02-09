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

func Test_SaveShortSellLendTransaction(t *testing.T) {
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

	if err := saveShortSellLendTransaction(transaction, db); err != nil {
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
		Id:            1,
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

func Test_SquareOffLends(t *testing.T) {
	makeAsk := func(userId uint32, stockId uint32, ot OrderType, stockQty uint64, price uint64) *Ask {
		return &Ask{
			UserId:        userId,
			StockId:       stockId,
			OrderType:     ot,
			StockQuantity: stockQty,
			Price:         price,
		}
	}

	user := &User{Id: 2, Cash: 2000}

	stocks := []*Stock{
		{Id: 1, CurrentPrice: 200},
		{Id: 2, CurrentPrice: 200},
	}

	ssbs := []*ShortSellBank{
		{
			StockId:         1,
			AvailableStocks: 100,
		},
		{
			StockId:         2,
			AvailableStocks: 100,
		},
	}

	db := getDB()

	defer func() {
		db.Exec("DELETE FROM OrderDepositTransactions")
		db.Exec("DELETE FROM Transactions")
		db.Exec("DELETE FROM ShortSellLends")
		db.Exec("DELETE FROM ShortSellBank")
		db.Exec("DELETE FROM Asks")
		db.Delete(ssbs)
		db.Delete(user)
		db.Delete(stocks)
	}()

	testCases := []*Ask{
		makeAsk(2, 1, Limit, 5, 200),
		makeAsk(2, 1, Limit, 5, 200),
		makeAsk(2, 1, Limit, 5, 200),
		makeAsk(2, 1, Limit, 5, 200),
		makeAsk(2, 2, Limit, 5, 200),
		makeAsk(2, 2, Limit, 5, 200),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	for _, stock := range stocks {
		if err := db.Create(stock).Error; err != nil {
			t.Fatal(err)
		}
	}

	LoadStocks()

	for _, ssb := range ssbs {
		if err := db.Create(ssb).Error; err != nil {
			t.Fatal(err)
		}
	}

	for _, ask := range testCases {
		if _, err := PlaceAskOrder(2, ask); err != nil {
			t.Fatal(err)
		}
	}

	if err := squareOffLends(); err != nil {
		t.Fatal(err)
	}

	stockOwned, err := GetStocksOwned(user.Id)

	if err != nil {
		t.Fatal(err)
	}

	if stockOwned[stocks[0].Id] != -20 {
		t.Fatalf("expected -20 got %d", stockOwned[stocks[0].Id])
	}

	if stockOwned[stocks[1].Id] != -10 {
		t.Fatalf("expected -10 got %d", stockOwned[stocks[1].Id])
	}

	for _, ssb := range ssbs {
		availableStocks, err := getAvailableLendStocks(ssb.StockId)

		if err != nil {
			t.Fatal(err)
		}

		if availableStocks != 100 {
			t.Fatalf("expected 100 got %d", availableStocks)
		}
	}
}

func Test_GetUserShortSellStocks(t *testing.T) {
	noOfStocks, err := getUserShortSellStocks(1, 1)

	if err != nil {
		t.Fatal(err)
	}

	if noOfStocks != 0 {
		t.Fatalf("expected 0 got %d", noOfStocks)
	}

	user := &User{Id: 1}
	stock := &Stock{Id: 1}
	db := getDB()

	defer func() {
		db.Delete(user)
		db.Delete(stock)
		db.Exec("DELETE FROM ShortSellLends")
	}()

	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(stock).Error; err != nil {
		t.Fatal(err)
	}

	ssl := []*ShortSellLends{
		{
			StockId:       1,
			UserId:        1,
			StockQuantity: 5,
		},
		{
			StockId:       1,
			UserId:        1,
			StockQuantity: 5,
		},
	}

	for _, lend := range ssl {
		if err := db.Save(lend).Error; err != nil {
			t.Fatal(err)
		}
	}

	noOfStocks, _ = getUserShortSellStocks(1, 1)

	if noOfStocks != 10 {
		t.Fatalf("expected 0 got %d", noOfStocks)
	}
}
