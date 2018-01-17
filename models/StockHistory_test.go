package models

import (
	"testing"
	"time"

	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestStockHistoryToProto(t *testing.T) {
	o := &StockHistory{
		StockId:   3,
		Close:     23,
		CreatedAt: "2017-02-09T00:00:00",
		Interval:  3,
		Open:      500,
		High:      1000,
		Low:       200,
	}

	o_proto := o.ToProto()

	if !testutils.AssertEqual(t, o, o_proto) {
		t.Fatal("Converted value not equal")
	}
}

func Test_RecordNMinuteOHLCs(t *testing.T) {
	t.Logf("Testing record N minute ohlc")
	//Fake time for doing minute manipulation
	fakeTime := time.Now()
	//Manipulate faked time such that it starts at the minute after a %5
	fakeTime = fakeTime.Add((-time.Duration(fakeTime.Minute()%5) + 1) * time.Minute)

	db, err := DbOpen()
	if err != nil {
		t.Fatalf("Error:Opening Database for inserting Stocks,Record failed +%v", err)
	}
	defer db.Close()

	stock := &Stock{
		Id:           1,
		CurrentPrice: 2000,
	}
	db.Save(stock)
	defer db.Delete(stock)

	stock1 := &Stock{
		Id:           2,
		CurrentPrice: 1500,
	}
	db.Save(stock1)
	defer db.Delete(stock1)

	LoadStocks()
	defer db.Exec("DELETE FROM StockHistory")

	for i := 0; i < 10; i++ {
		//Update StockPrice with multiples of i
		UpdateStockPrice(1, uint32(250*i))
		recordOneMinuteOHLC(db, fakeTime.Add(time.Duration(i)*time.Minute))
		//Check if minute is a multiple of 5
		if fakeTime.Add(time.Duration(i)*time.Minute).Minute()%5 == 0 {
			var retrievedHistories []StockHistory
			//Get relevant entries from db (Similar to the recordHigherOHLC)
			dbWhere := db.Where("intervalRecord = ? AND stockId = ? AND createdAt >= ?", 1, 1, fakeTime.UTC().Format(time.RFC3339))
			dbWhere.Order("createdAt desc").Limit(5).Find(&retrievedHistories)

			t.Logf("No of such entries found %v", len(retrievedHistories))

			err := recordNMinuteOHLC(db, 1, retrievedHistories, 5, fakeTime.Add(time.Duration(i)*time.Minute))

			if err != nil {
				t.Fatalf("Recording NMinute interval failed %v", err)
			}
		}
	}

	if err != nil {
		t.Fatalf("Recording one minute interval failed with the error +%v", err)
	}

	//Get recordings with interval 5 of stockId 1
	var retrievedHistory []*StockHistory
	db.Where("stockId = 1 AND intervalRecord = ?", 5).Find(&retrievedHistory)

	//stockId 1 should only have 2 entries over a range of 10 minutes
	if len(retrievedHistory) != 2 {
		for _, v := range retrievedHistory {
			t.Logf("%v", v)
		}
		t.Fatalf("Expected 1 history entries for stockId 1 but got +%v", len(retrievedHistory))
	}

	//Adding 4 minutes to initial fake time gives you the first multiple of 5
	expectedStock := &StockHistory{StockId: 1, Open: 2000, High: 2000, Interval: 5, Low: 0, Close: 1000, CreatedAt: fakeTime.Add(4 * time.Minute).UTC().Format(time.RFC3339)}
	if !testutils.AssertEqual(t, retrievedHistory[0], expectedStock) {
		t.Fatalf("Expected %+v got %+v", expectedStock, retrievedHistory[0])
	}

	//Adding 9 minutes to initial fake time gives you the second multiple of 5
	expectedStock = &StockHistory{StockId: 1, Open: 1000, High: 2250, Interval: 5, Low: 1000, Close: 2250, CreatedAt: fakeTime.Add(9 * time.Minute).UTC().Format(time.RFC3339)}
	if !testutils.AssertEqual(t, retrievedHistory[1], expectedStock) {
		t.Fatalf("Expected %+v got %+v", expectedStock, retrievedHistory[1])
	}
}
func Test_RecordOneMinuteOHLC(t *testing.T) {
	t.Logf("Testing record one minute ohlc")

	fakeTime := time.Now()
	db, err := DbOpen()
	if err != nil {
		t.Fatalf("Error:Opening Database for inserting Stocks,Record failed +%v", err)
	}
	defer db.Close()
	defer db.Exec("DELETE FROM Stocks")
	stock := &Stock{
		Id:           1,
		CurrentPrice: 2000,
	}
	db.Save(stock)
	defer db.Delete(stock)
	stock1 := &Stock{
		Id:           2,
		CurrentPrice: 1500,
	}
	db.Save(stock1)
	defer db.Delete(stock1)

	LoadStocks()

	UpdateStockPrice(1, 2500)
	UpdateStockPrice(2, 500)
	UpdateStockPrice(2, 1500)
	UpdateStockPrice(1, 900)
	err = recordOneMinuteOHLC(db, fakeTime)

	fakeTime1 := fakeTime.Add(time.Minute)
	UpdateStockPrice(1, 300)
	UpdateStockPrice(2, 2500)
	UpdateStockPrice(1, 500)
	UpdateStockPrice(2, 600)
	err = recordOneMinuteOHLC(db, fakeTime1)

	if err != nil {
		t.Fatalf("Recording one minute interval failed with the error +%v", err)
	}
	defer db.Exec("DELETE FROM StockHistory")

	var retrievedHistory []*StockHistory
	db.Where("stockId = 1").Find(&retrievedHistory)

	expectedStock := &StockHistory{StockId: 1, Open: 2000, High: 2500, Interval: 1, Low: 900, Close: 900, CreatedAt: fakeTime.UTC().Format(time.RFC3339)}

	if len(retrievedHistory) != 2 {
		t.Fatalf("Expected 2 history entries for stockId 1 but got +%v", len(retrievedHistory))
	}

	if !testutils.AssertEqual(t, retrievedHistory[0], expectedStock) {
		t.Fatalf("Expected %+v got %+v", expectedStock, retrievedHistory[0])
	}

	db.Where("stockId = 2").Find(&retrievedHistory)

	expectedStock = &StockHistory{StockId: 2, Open: 1500, High: 2500, Low: 600, Interval: 1, Close: 600, CreatedAt: fakeTime1.UTC().Format(time.RFC3339)}

	if !testutils.AssertEqual(t, retrievedHistory[1], expectedStock) {
		t.Fatalf("Expected %+v got %+v", expectedStock, retrievedHistory[1])
	}
}
func Test_GetStockHistory(t *testing.T) {
	t.Logf("Testing getstockhistory")
	var stock = &Stock{Id: 1, CurrentPrice: 2000}
	now := time.Now()
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

	for i := 0; i <= 75; i++ {
		UpdateStockPrice(1, uint32(i*20))
		recordOneMinuteOHLC(db, now.Add(time.Minute*time.Duration(i)))
	}
	retrievedHistories, err := GetStockHistory(1, 1)
	if err != nil {
		t.Fatalf("GetStockHistory Errorred +%v", err)
	}
	if len(retrievedHistories) != TIMES_RESOLUTION {
		t.Fatalf("Expected %v histories but got %v", len(retrievedHistories), TIMES_RESOLUTION)
	}
	expectedStockHistory := &StockHistory{StockId: 1, Close: 680, CreatedAt: now.Add(34 * time.Minute).UTC().Format(time.RFC3339), Interval: 1, Open: 660, High: 680, Low: 660}
	if !testutils.AssertEqual(t, expectedStockHistory, retrievedHistories[(75-34)]) {
		t.Fatalf("Expected %v but got %v", expectedStockHistory, retrievedHistories[75-34])
		// 0th index would have 75*20 hence for 34*20  0+(75-34) should pass
	}
}
