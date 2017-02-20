package models

import (
	"time"
	"testing"
	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils/test"
)

func TestLeaderboardRowToProto(t *testing.T) {
	lr := &LeaderboardRow{
		Id:         2,
		UserId:     5,
		Rank:       1,
		Cash:       1000,
		Debt:       10,
		StockWorth: -50,
		TotalWorth: -300,
	}

	lr_proto := lr.ToProto()

	if !testutils.AssertEqual(t, lr, lr_proto) {
		t.Fatal("Converted values not equal!")
	}
}

func Test_UpdateLeaderboard(t *testing.T) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Test_UpdateLeaderboard",
	})

	var makeTrans = func(userId uint32, stockId uint32, transType TransactionType, stockQty int32, price uint32, total int32) *Transaction {
		return &Transaction{
			UserId:        userId,
			StockId:       stockId,
			Type:          transType,
			StockQuantity: stockQty,
			Price:         price,
			Total:         total,
		}
	}

	var makeUser = func(id uint32, email string, name string, cash uint32, total int32) *User {
		return &User{
		  	Id : id,
			Email : email,
			Name : name,
			Cash : cash,
			Total : total,
			CreatedAt : time.Now().Format(time.RFC3339),
		}
	}

	var makeStock = func(id uint32, sName string, fName string, desc string, curPrice uint32, dayHigh uint32, dayLow uint32, allHigh uint32, allLow uint32, stocks uint32, upOrDown bool) *Stock {
		return &Stock{
		  	Id: id,
			ShortName: sName,
			FullName: fName,
			Description: desc,
			CurrentPrice: curPrice,
			DayHigh: dayHigh,
			DayLow: dayLow,
			AllTimeHigh: allHigh,
			AllTimeLow: allLow,
			StocksInExchange: stocks,
			UpOrDown: upOrDown,
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
	}

	users := []*User{
		makeUser(2, "test@testmail.com", "Test", 100000, 100000),
		makeUser(3, "lol@lol.com", "Rick", 100000, 100000),
		makeUser(4, "noob@noob.com", "Deep", 100000, 100000),
		makeUser(5, "haha@haha.com", "CR7", 100000, 100000),
	}

	stocks := []*Stock{
		makeStock(1, "FB", "Facebook", "Social", 100, 200, 60, 300, 10, 2000, true),
		makeStock(2, "MS", "Microsoft", "MS Corp", 300, 450, 60, 600, 10, 2000, true),
	}

	transactions := []*Transaction{
		makeTrans(2, 1, OrderFillTransaction, 10, 200, -2000),
		makeTrans(2, 2, OrderFillTransaction, -15, 300, 4500),
		makeTrans(3, 1, OrderFillTransaction, 40, 200, -8000),
		makeTrans(3, 2, OrderFillTransaction, -30, 300, 9000),
		makeTrans(4, 1, OrderFillTransaction, -10, 200, 2000),
		makeTrans(4, 2, OrderFillTransaction, 15, 300, -4500),
		makeTrans(5, 1, OrderFillTransaction, 40, 200, -8000),
		makeTrans(5, 2, OrderFillTransaction, -30, 300, 9000),
	}

	var results []leaderboardQueryData
	var leaderboardEntries []*LeaderboardRow

	db, err := DbOpen()
	if err != nil {
		t.Fatal("Failed opening DB to insert dummy data")
	}

	defer func() {

		db.Exec("TRUNCATE TABLE Leaderboard")
		for _, tr := range transactions {
			db.Delete(tr)
		}
		for _, stock := range stocks {
			db.Delete(stock)
		}
		for _, user := range users {
			db.Delete(user)
		}
		db.Close()
	}()

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}

	for _, stock := range stocks {
		if err := db.Create(stock).Error; err != nil {
			t.Fatal(err)
		}
	}

	for _, tr := range transactions {
		if err := db.Create(tr).Error; err != nil {
			t.Fatal(err)
		}
	}	

	db.Raw("SELECT U.id as user_id, U.cash as cash, SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed)) AS stock_worth, (U.cash + SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed))) AS total from Users U, Transactions T, Stocks S WHERE U.id = T.userId and T.stockId = S.id GROUP BY U.id ORDER BY Total DESC").Scan(&results)

	var rank = 1
	var counter = 1

	for index, result := range results {
		leaderboardEntries = append(leaderboardEntries, &LeaderboardRow{
			Id:         uint32(index + 1),
			UserId:     result.UserId,
			Cash:       result.Cash,
			Rank:       uint32(rank),
			Debt:       0,
			StockWorth: result.StockWorth,
			TotalWorth: result.Total,
		})

		counter += 1
		if index+1 < len(results) && results[index+1].Total < result.Total {
			rank = counter
		}
	}

	l.Infof("%+v", results)
	l.Infof("%+v", leaderboardEntries)	

	db.Exec("LOCK TABLES Leaderboard WRITE")
	defer db.Exec("UNLOCK TABLES")


	//begin transaction
	tx := db.Begin()
	
	db.Exec("TRUNCATE TABLE Leaderboard")

	for _, leaderboardEntry := range leaderboardEntries {
		if err := db.Save(leaderboardEntry).Error; err != nil {
			tx.Rollback()
			t.Fatalf("Error updating leaderboard. Failing. %+v", err)
			return
		}
	}

	//commit transaction
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("Error committing leaderboardUpdate transaction. Failing. %+v", err)
		return
	}

	l.Infof("Successfully updated leaderboard")
}

