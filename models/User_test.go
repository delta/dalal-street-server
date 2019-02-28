package models

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	testutils "github.com/delta/dalal-street-server/utils/test"
	"github.com/jinzhu/gorm"
)

func Test_Login(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	//Tests case for first time pragyan login
	httpmock.RegisterResponder("POST", "https://api.pragyan.org/19/event/login", httpmock.NewStringResponder(200, `{"status_code":200,"message": { "user_id": 2, "user_fullname": "TestName" , "user_name":"UserName", "user_country":"India" }}`))

	u, err := Login("test@testmail.com", "password")
	if err != nil {
		t.Fatalf("Login returned an error: %s", err)
	}

	defer func() {
		db := getDB()
		db.Delete(&Registration{UserId: u.Id})
		db.Delete(u)
	}()

	exU := User{
		Id:        u.Id,
		Email:     "test@testmail.com",
		Name:      "TestName",
		Cash:      STARTING_CASH,
		Total:     STARTING_CASH,
		CreatedAt: u.CreatedAt,
		IsHuman:   true,
	}
	if reflect.DeepEqual(u, exU) != true {
		t.Fatalf("Expected Login to return %+v, instead, got %+v", exU, u)
	}
	_, err = Login("test@testmail.com", "TestName")
	if err != nil {
		t.Fatalf("Login failed: '%s'", err)
	}

	//The email should be Registrationed with the previous login attempt
	u, _ = Login("test@testmail.com", "password")

	if reflect.DeepEqual(u, exU) != true {
		t.Fatalf("Expected Login to return %+v, instead, got %+v", exU, u)
	}
	_, err = Login("test@testmail.com", "TestName")
	if err != nil {
		t.Fatalf("Login failed: '%s'", err)
	}
	//allErrors, ok = migrate.DownSync(connStr, "../migrations")
}

func Test_Regsiter(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	//Tests case for first time pragyan login
	httpmock.RegisterResponder("POST", "https://api.pragyan.org/19/event/login", httpmock.NewStringResponder(200, `{"status_code":200,"message": { "user_id": 2, "user_fullname": "TestName" , "user_name":"UserName", "user_country":"India"}}`))
	err := RegisterUser("test@testname.com", "password", "FullName")
	defer func() {
		db := getDB()
		db.Exec("DELETE FROM Registrations")
		db.Exec("DELETE FROM Users")
	}()
	if err != AlreadyRegisteredError {
		t.Fatalf("Expected %+v but got %+v", AlreadyRegisteredError, err)
	}
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder("POST", "https://api.pragyan.org/19/event/login", httpmock.NewStringResponder(401, `{"status_code":401,"message": "Invalid Credentials"}`))
	err = RegisterUser("test@testname.com", "password", "FullName")

	if err != AlreadyRegisteredError {
		t.Fatalf("Expected %+v but got %+v", AlreadyRegisteredError, err)
	}
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder("POST", "https://api.pragyan.org/19/event/login", httpmock.NewStringResponder(400, `{"status_code":400,"message": "Account Not Registered"}`))
	err = RegisterUser("test@testname.com", "password", "FullName")
	db := getDB()
	registeredTestUser := &Registration{
		Email: "test@testname.com",
	}
	if err != nil {
		t.Fatalf("Expected %+v but got %+v", nil, err)
	}
	err = db.Find(registeredTestUser).Error
	if !checkPasswordHash("password", registeredTestUser.Password) {
		t.Fatalf("Incorrect password")
	}

	expectedKey, _ := getVerificationKey("test@testname.com")
	expectedUser := &Registration{
		Id:              registeredTestUser.Id,
		UserId:          registeredTestUser.UserId,
		Email:           "test@testname.com",
		Password:        registeredTestUser.Password,
		Name:            "FullName",
		IsPragyan:       false,
		IsVerified:      false,
		VerificationKey: expectedKey,
	}
	if !testutils.AssertEqual(t, expectedUser, registeredTestUser) {
		t.Fatalf("Expected %+v but got %+v", expectedUser, registeredTestUser)
	}
	if err != nil {
		t.Fatalf("Retrieving from db failed with %v", err)
	}

}

func TestUserToProto(t *testing.T) {
	o := &User{
		Id:           2,
		Email:        "test@testmail.com",
		Name:         "test user",
		Cash:         10000,
		Total:        -200,
		CreatedAt:    "2017-06-08T00:00:00",
		IsHuman:      true,
		ReservedCash: 10,
	}

	oProto := o.ToProto()

	if !testutils.AssertEqual(t, o, oProto) {
		t.Fatalf("Converted values not equal!+%v,+%v", o, oProto)
	}

}

func Test_PlaceAskOrder(t *testing.T) {
	var makeTrans = func(userId uint32, stockId uint32, transType TransactionType, stockQty int64, price uint64, total int64) *Transaction {
		return &Transaction{
			UserId:        userId,
			StockId:       stockId,
			Type:          transType,
			StockQuantity: stockQty,
			Price:         price,
			Total:         total,
		}
	}

	var makeAsk = func(userId uint32, stockId uint32, ot OrderType, stockQty uint64, price uint64) *Ask {
		return &Ask{
			UserId:        userId,
			StockId:       stockId,
			OrderType:     ot,
			StockQuantity: stockQty,
			Price:         price,
		}
	}

	var user = &User{Id: 2}
	var stock = &Stock{Id: 1, CurrentPrice: 200}

	transactions := []*Transaction{
		makeTrans(2, 1, FromExchangeTransaction, 10, 200, 2000),
		makeTrans(2, 1, FromExchangeTransaction, -10, 200, 2000),
		makeTrans(2, 1, FromExchangeTransaction, -10, 200, 2000),
	}

	testcases := []struct {
		ask  *Ask
		pass bool
	}{
		{makeAsk(2, 1, Limit, 5, 200), true},
		{makeAsk(2, 1, Limit, 2, 200), true},
		{makeAsk(2, 1, Limit, 3, 200), true},
		{makeAsk(2, 1, Limit, 11, 200), false},
		{makeAsk(2, 1, Limit, 11, 2000), false}, // too high a price won't be allowed
		{makeAsk(2, 1, Limit, 10, 2000), false}, // with transaction fee, not enough cash
		{makeAsk(2, 1, Limit, 11, 2), false},    // too low a price won't be allowed
	}

	db := getDB()
	defer func() {
		for _, tr := range transactions {
			db.Delete(tr)
		}
		for _, tc := range testcases {
			db.Delete(tc.ask)
		}
		db.Exec("DELETE FROM OrderDepositTransactions")
		db.Exec("DELETE FROM Transactions") // Because we create additional OrderFee Transactions
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
		db.Delete(user)

		delete(userLocks.m, 2)
	}()

	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(stock).Error; err != nil {
		t.Fatal(err)
	}
	LoadStocks()

	for _, tr := range transactions {
		if err := db.Create(tr).Error; err != nil {
			t.Fatal(err)
		}
	}

	wg := sync.WaitGroup{}
	fm := sync.Mutex{}
	for _, tc := range testcases {
		if tc.pass != true {
			continue
		}
		tc := tc
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := PlaceAskOrder(2, tc.ask)

			if err != nil {
				fm.Lock()
				defer fm.Unlock()
				t.Fatalf("Did not expect error. Got %+v", err)
			}

			a := &Ask{}
			db.First(a, tc.ask.Id)
			fm.Lock()
			defer fm.Unlock()
			if !testutils.AssertEqual(t, a, tc.ask) {
				t.Fatalf("Got %+v; Want %+v;", a, tc.ask)
			}
		}()
	}

	wg.Wait()

	tid, terr := PlaceAskOrder(2, testcases[len(testcases)-2].ask)
	if terr == nil {
		t.Fatalf("Did not expect success. Failing %+v %+v", tid, terr)
	}

	id, err := PlaceAskOrder(2, testcases[len(testcases)-1].ask)
	if err == nil {
		t.Fatalf("Did not expect success. Failing %+v %+v", id, err)
	}
}

func Test_PlaceBidOrder(t *testing.T) {
	var makeTrans = func(userId uint32, stockId uint32, transType TransactionType, stockQty int64, price uint64, total int64) *Transaction {
		return &Transaction{
			UserId:        userId,
			StockId:       stockId,
			Type:          transType,
			StockQuantity: stockQty,
			Price:         price,
			Total:         total,
		}
	}

	var makeBid = func(userId uint32, stockId uint32, ot OrderType, stockQty uint64, price uint64) *Bid {
		return &Bid{
			UserId:        userId,
			StockId:       stockId,
			OrderType:     ot,
			StockQuantity: stockQty,
			Price:         price,
		}
	}

	var updateUserCash = func(user *User, cash uint64, db *gorm.DB) {
		user.Cash = cash
		db.Save(user)
	}

	var user = &User{Id: 2, Cash: 2000, ReservedCash: 0}
	var stock = &Stock{Id: 1, CurrentPrice: 200}

	transactions := []*Transaction{
		makeTrans(2, 1, FromExchangeTransaction, 10, 200, 2000),
		makeTrans(2, 1, FromExchangeTransaction, -10, 200, 2000),
		makeTrans(2, 1, FromExchangeTransaction, -10, 200, 2000),
	}

	testcases := []struct {
		bid  *Bid
		pass bool
	}{
		{makeBid(2, 1, Limit, 5, 200), true},
		{makeBid(2, 1, Limit, 2, 200), true},
		{makeBid(2, 1, Limit, 2, 200), true},
		{makeBid(2, 1, Limit, 11, 200), false},
		{makeBid(2, 1, Limit, 11, 2000), false}, // too high a price won't be allowed
		{makeBid(2, 1, Limit, 10, 200), false},  // with transaction fee, cash wont be enough
		{makeBid(2, 1, Limit, 11, 2), false},    // too low a price won't be allowed
	}

	db := getDB()
	defer func() {
		for _, tr := range transactions {
			db.Delete(tr)
		}
		for _, tc := range testcases {
			db.Delete(tc.bid)
		}
		db.Exec("DELETE FROM OrderDepositTransactions")
		db.Exec("DELETE FROM Transactions") // Cuz we add new OrderFeeTransaction
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
		db.Delete(user)

		delete(userLocks.m, 2)
	}()

	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(stock).Error; err != nil {
		t.Fatal(err)
	}
	LoadStocks()

	for _, tr := range transactions {
		if err := db.Create(tr).Error; err != nil {
			t.Fatal(err)
		}
	}

	wg := sync.WaitGroup{}
	fm := sync.Mutex{}

	for _, tc := range testcases {
		if tc.pass != true {
			continue
		}
		tc := tc
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := PlaceBidOrder(2, tc.bid)
			if err != nil {
				fm.Lock()
				defer fm.Unlock()
				t.Fatalf("Did not expect error. Bid: %+v. Error: %+v", tc.bid, err)
			}

			b := &Bid{}
			db.First(b, tc.bid.Id)
			fm.Lock()
			defer fm.Unlock()
			if !testutils.AssertEqual(t, b, tc.bid) {
				t.Fatalf("Got %+v; Want %+v;", b, tc.bid)
			}
		}()
	}

	wg.Wait()

	updateUserCash(user, 2000, db)
	tid, terr := PlaceBidOrder(2, testcases[len(testcases)-2].bid)
	if terr == nil {
		t.Fatalf("Did not expect success. Failing %+v %+v", tid, terr)
	}

	updateUserCash(user, 2000, db)
	id, err := PlaceBidOrder(2, testcases[len(testcases)-1].bid)
	if err == nil {
		t.Fatalf("Did not expect success. Failing %+v %+v", id, err)
	}
}

func Test_CancelOrder(t *testing.T) {
	var makeAsk = func(userId uint32, askId uint32, stockId uint32, ot OrderType, stockQty uint64, price uint64) *Ask {
		return &Ask{
			Id:            askId,
			UserId:        userId,
			StockId:       stockId,
			OrderType:     ot,
			StockQuantity: stockQty,
			Price:         price,
		}
	}

	var makeBid = func(userId uint32, bidId uint32, stockId uint32, ot OrderType, stockQty uint64, price uint64) *Bid {
		return &Bid{
			Id:            bidId,
			UserId:        userId,
			StockId:       stockId,
			OrderType:     ot,
			StockQuantity: stockQty,
			Price:         price,
		}
	}

	var user = &User{Id: 2, Cash: 3000, ReservedCash: 0}
	var stock = &Stock{Id: 1}

	var bids = []*Bid{
		makeBid(2, 150, 1, Limit, 5, 200),
		makeBid(2, 160, 1, Limit, 2, 200),
	}
	var asks = []*Ask{
		makeAsk(2, 150, 1, Limit, 5, 200),
		makeAsk(2, 160, 1, Limit, 2, 200),
	}

	testcases := []struct {
		userId  uint32
		orderId uint32
		isAsk   bool
		pass    bool
	}{
		{2, 150, false, true},
		{2, 160, false, true},
		{3, 150, false, false},
		{2, 250, false, false},
		{2, 150, true, true},
		{2, 160, true, true},
		{1, 150, false, false},
		{2, 260, false, false},
	}

	db := getDB()
	defer func() {
		for _, a := range asks {
			db.Delete(a)
		}
		for _, b := range bids {
			db.Delete(b)
		}
		db.Exec("DELETE FROM OrderDepositTransactions")
		db.Exec("DELETE FROM Transactions")
		db.Exec("DELETE FROM StockHistory")
		db.Delete(stock)
		db.Delete(user)

		delete(userLocks.m, 2)
	}()

	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(stock).Error; err != nil {
		t.Fatal(err)
	}

	for _, a := range asks {
		if _, err := PlaceAskOrder(2, a); err != nil {
			t.Fatal(err)
		}
	}

	for _, b := range bids {
		if _, err := PlaceBidOrder(2, b); err != nil {
			t.Fatal(err)
		}
	}

	wg := sync.WaitGroup{}
	fm := sync.Mutex{}

	for _, tc := range testcases {
		tc := tc
		wg.Add(1)
		go func() {
			defer wg.Done()
			askOrder, bidOrder, err := CancelOrder(tc.userId, tc.orderId, tc.isAsk)
			if tc.pass == true {
				if err != nil {
					fm.Lock()
					defer fm.Unlock()
					t.Fatalf("Did not expect error. Got %+v", err)
				} else if tc.isAsk && (askOrder == nil || bidOrder != nil) {
					fm.Lock()
					defer fm.Unlock()
					t.Fatalf("For tc.isAsk, only askOrder should be not-nil")
				} else if !tc.isAsk && (askOrder != nil || bidOrder == nil) {
					fm.Lock()
					defer fm.Unlock()
					t.Fatalf("For !tc.isAsk, only bidOrder should be not-nil")
				}
			} else if tc.pass == false && err == nil {
				fm.Lock()
				defer fm.Unlock()
				t.Fatalf("Expected error. Didn't get it. Failing.")
			}
		}()
	}

	wg.Wait()

}

func Test_GetStocksOwned(t *testing.T) {
	var makeTrans = func(userId uint32, stockId uint32, transType TransactionType, stockQty int64, price uint64, total int64) *Transaction {
		return &Transaction{
			UserId:        userId,
			StockId:       stockId,
			Type:          transType,
			StockQuantity: stockQty,
			Price:         price,
			Total:         total,
		}
	}

	users := []*User{
		{Id: 2, Email: "a@b.com", Cash: 2000},
		{Id: 3, Email: "c@d.com", Cash: 1000},
		{Id: 4, Email: "e@f.com", Cash: 5000},
	}

	stocks := []*Stock{
		{Id: 1},
		{Id: 2},
		{Id: 3},
	}

	transactions := []*Transaction{
		makeTrans(2, 1, FromExchangeTransaction, 10, 1, 2000),
		makeTrans(2, 1, FromExchangeTransaction, 10, 2, 2000),
		makeTrans(2, 2, FromExchangeTransaction, -10, 1, 2000),

		makeTrans(3, 1, FromExchangeTransaction, 10, 1, 2000),
		makeTrans(3, 3, FromExchangeTransaction, -10, 2, 2000),

		makeTrans(4, 2, FromExchangeTransaction, -10, 2, 2000),
		makeTrans(4, 2, FromExchangeTransaction, 10, 1, 2000),
		makeTrans(4, 2, FromExchangeTransaction, -10, 1, 2000),
		makeTrans(4, 3, FromExchangeTransaction, 10, 1, 2000),
	}

	testcases := []struct {
		userId   uint32
		expected map[uint32]int32
	}{
		{userId: 2, expected: map[uint32]int32{1: 20, 2: -10}},
		{userId: 3, expected: map[uint32]int32{1: 10, 3: -10}},
		{userId: 4, expected: map[uint32]int32{2: -10, 3: 10}},
	}

	db := getDB()
	defer func() {
		for _, tr := range transactions {
			if err := db.Delete(tr).Error; err != nil {
				t.Fatal(err)
			}
		}
		db.Exec("DELETE FROM StockHistory")
		for _, stock := range stocks {
			if err := db.Delete(stock).Error; err != nil {
				t.Fatal(err)
			}
		}
		for _, user := range users {
			if err := db.Delete(user).Error; err != nil {
				t.Fatal(err)
			}
			delete(userLocks.m, user.Id)
		}
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

	wg := sync.WaitGroup{}
	fm := sync.Mutex{}

	for _, tc := range testcases {
		tc := tc
		wg.Add(1)
		go func() {
			defer wg.Done()
			ret, err := GetStocksOwned(tc.userId)
			fm.Lock()
			defer fm.Unlock()

			if err != nil {
				t.Fatalf("Did not expect error. Got %+v", err)
			}
			if !testutils.AssertEqual(t, tc.expected, ret) {
				t.Fatalf("Got %+v; want %+v", ret, tc.expected)
			}
		}()
	}

	wg.Wait()
}

func Test_PerformBuyFromExchangeTransaction(t *testing.T) {
	origCash := map[uint32]uint64{2: 2000, 3: 1000, 4: 5000}

	users := []*User{
		{Id: 2, Email: "a@b.com", Cash: origCash[2]},
		{Id: 3, Email: "c@d.com", Cash: origCash[3]},
		{Id: 4, Email: "e@f.com", Cash: origCash[4]},
	}

	stocks := []*Stock{
		{Id: 1, StocksInExchange: 30, CurrentPrice: 100},
		{Id: 2, StocksInExchange: 40, CurrentPrice: 500},
		{Id: 3, StocksInExchange: 20, CurrentPrice: 200},
	}

	stockPrices := []struct {
		stockId uint32
		price   uint64
	}{
		{1, 101},
		{2, 498},
		{2, 499},
		{3, 201},
		{1, 102},
		{2, 500},
		{3, 200},
	}

	testcases := []struct {
		userId           uint32
		stockId          uint32
		stockQuantity    uint64
		maxStkQtyGot     uint64
		buyLimitExceeded bool
	}{
		{2, 1, 10, 10, false},
		{3, 1, 5, 5, false},
		{4, 1, 30, 30, true},

		{2, 2, 15, 4, false},
		{3, 2, 10, 2, false},
		{4, 2, 20, 10, false},

		{2, 3, 7, 10, false},
		{3, 3, 8, 5, false},
		{4, 3, 10, 25, false},
	}

	type lockedTrList struct {
		sync.Mutex
		trlist []*Transaction
	}
	transactions := struct {
		sync.Mutex
		m map[uint32]*lockedTrList
	}{m: make(map[uint32]*lockedTrList)}
	transactions.m[2] = &lockedTrList{}
	transactions.m[3] = &lockedTrList{}
	transactions.m[4] = &lockedTrList{}

	db := getDB()
	defer func() {
		for _, ltrlist := range transactions.m {
			for _, tr := range ltrlist.trlist {
				if err := db.Delete(tr).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
		db.Exec("DELETE FROM TransactionSummary")
		db.Exec("DELETE FROM StockHistory")
		for _, stock := range stocks {
			if err := db.Delete(stock).Error; err != nil {
				t.Fatal(err)
			}
		}
		for _, user := range users {
			if err := db.Delete(user).Error; err != nil {
				t.Fatal(err)
			}
			delete(userLocks.m, user.Id)
		}
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

	if err := LoadStocks(); err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	fm := sync.Mutex{}

	for _, tc := range testcases {
		tc := tc
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr, err := PerformBuyFromExchangeTransaction(tc.userId, tc.stockId, tc.stockQuantity)

			fm.Lock()
			defer fm.Unlock()

			if err != nil {
				if _, ok := err.(NotEnoughCashError); ok {
					u, err1 := GetUserCopy(tc.userId)
					if err1 != nil {
						t.Fatalf("Error getting latest user data %+v", err1)
					}
					t.Logf("Got not enough cash error, current cash %d, tc: %+v. err: %+v", u.Cash, tc, err)
					return
				}
				if _, ok := err.(BuyLimitExceededError); ok {
					t.Logf("Got buy limit exceeded error, current buy limit : %d, orderQuantity : %d, didExceed : %v", BUY_LIMIT, tc.stockQuantity, tc.buyLimitExceeded)
					return
				}
				if _, ok := err.(NotEnoughStocksError); ok {
					allStocks.m[tc.stockId].RLock()
					sie := allStocks.m[tc.stockId].stock.StocksInExchange
					allStocks.m[tc.stockId].RUnlock()

					if sie == 0 {
						t.Logf("Got Not enough stocks error, stocks in exchange got empty")
						return
					}
				}
				t.Fatalf("Did not expect error. Got %+v", err)
			}

			// append the transaction to the user's list of transactions
			ltrList := transactions.m[tc.userId]
			ltrList.Lock()
			ltrList.trlist = append(ltrList.trlist, tr)
			ltrList.Unlock()

			if tr.StockQuantity < 0 {
				t.Fatalf("Got negative! Wut. Got %+v", tr.StockQuantity)
			}
			if tc.maxStkQtyGot < uint64(tr.StockQuantity) {
				t.Fatalf("Got more than possible. Allowed %+v; Got %+v", tc.maxStkQtyGot, tr.StockQuantity)
			}
		}()
	}

	for _, sp := range stockPrices {
		wg.Add(1)
		sid := sp.stockId
		sprice := sp.price
		go func() {
			defer wg.Done()

			allStocks.m[sid].Lock()
			allStocks.m[sid].stock.CurrentPrice = sprice
			allStocks.m[sid].Unlock()
		}()
	}

	wg.Wait()

	// verify the cash for each user
	for uid, oc := range origCash {
		newCash := int64(oc)
		for _, tr := range transactions.m[uid].trlist {
			newCash += tr.Total
		}
		u, err := GetUserCopy(uid)
		if err != nil {
			t.Fatalf("Error getting latest user data %+v", err)
		}
		if uint64(newCash) != u.Cash {
			t.Fatalf("User %d's cash not consistent. Got %d; want %d", uid, u.Cash, newCash)
		}
	}
}

// Helper for the deadlock test written below. If you want to test any table OTHER than TransactionSummary,
// all you need to do is update this helper function. The actual Test_Deadlock function need not be updated.
func saveTransSum(userID int, t *testing.T) {
	db := getDB()
	tx := db.Begin()

	transactionSummary := &TransactionSummary{
		UserId:        uint32(userID),
		StockId:       2,
		StockQuantity: 10,
		Price:         12.5,
	}

	if err := tx.Save(transactionSummary).Error; err != nil {
		t.Errorf("Error updating the transaction summary. Rolling back. Error : +%v", err)
		tx.Rollback()
		return
	}

	if err := tx.Commit().Error; err != nil {
		t.Errorf("Error committing the transaction. Failing. %+v", err)
		tx.Rollback()
		return
	}

	t.Logf("TransactionSummary table updated successfully - %+v", transactionSummary)
}

// This is a test for deadlock when multiple goroutines spam insert statements into the TransactionSummary table.
// However, with a few modifications, it can be made to test any table for deadlock possibilities. Feel free to modify
// this test if there's any other table you'd like to test insert statements on.
func Test_Deadlock(t *testing.T) {

	db := getDB()
	numUsers := 10

	for i := 1; i <= numUsers; i++ {
		user := &User{Id: uint32(i), Email: fmt.Sprintf("user%v@deadlock.com", i), Cash: 1000000}
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		} else {
			t.Logf("Created user %v", i)
		}
	}

	stock := &Stock{Id: 2, StocksInExchange: 30, CurrentPrice: 100}
	if err := db.Create(stock).Error; err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Exec("DELETE FROM TransactionSummary")
		db.Exec("DELETE FROM Users")
		db.Exec("DELETE FROM Stocks")
	}()

	wg := &sync.WaitGroup{}

	for i := 1; i <= numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			saveTransSum(userID, t)
		}(i)
	}

	wg.Wait()
}

func Test_PerformMortgageTransaction(t *testing.T) {
	var makeTrans = func(userId uint32, stockId uint32, transType TransactionType, stockQty int64, price uint64, total int64) *Transaction {
		return &Transaction{
			UserId:        userId,
			StockId:       stockId,
			Type:          transType,
			StockQuantity: stockQty,
			Price:         price,
			Total:         total,
		}
	}

	users := []*User{
		{Id: 2, Email: "a@b.com", Cash: 2000},
		{Id: 3, Email: "c@d.com", Cash: 3000},
		{Id: 4, Email: "e@f.com", Cash: 4000},
	}

	stocks := []*Stock{
		{Id: 1, CurrentPrice: 100, LastTradePrice: 100},
		{Id: 2, CurrentPrice: 500, LastTradePrice: 500},
		{Id: 3, CurrentPrice: 200, LastTradePrice: 200},
	}

	transactions := []*Transaction{
		makeTrans(2, 1, FromExchangeTransaction, 20, 100, 2000),
		makeTrans(2, 1, FromExchangeTransaction, -10, 100, 2000),
		makeTrans(2, 2, FromExchangeTransaction, -10, 100, 2000),
		makeTrans(2, 3, FromExchangeTransaction, 20, 200, 2000),

		makeTrans(4, 1, FromExchangeTransaction, 5, 100, 2000),
	}

	testcases := []struct {
		userId        uint32
		stockId       uint32
		stockQuantity int64
		cashGained    uint64
		stockLeft     int64
		retrievePrice uint64
	}{
		{2, 1, -7, 7 * 100 * MORTGAGE_DEPOSIT_RATE / 100, 3, 100},
		{2, 1, -2, 2 * 100 * MORTGAGE_DEPOSIT_RATE / 100, 1, 100},
		{2, 3, -15, 15 * 200 * MORTGAGE_DEPOSIT_RATE / 100, 5, 200},
		{2, 3, 5, 5 * 200 * MORTGAGE_RETRIEVE_RATE / 100, 10, 200},
		{2, 1, 5, 5 * 100 * MORTGAGE_RETRIEVE_RATE / 100, 6, 100},
	}

	db := getDB()
	defer func() {
		for _, tr := range transactions {
			if err := db.Delete(tr).Error; err != nil {
				t.Fatal(err)
			}
		}
		db.Exec("DELETE FROM StockHistory")
		db.Exec("DELETE FROM MortgageDetails")
		for _, stock := range stocks {
			if err := db.Delete(stock).Error; err != nil {
				t.Fatal(err)
			}
		}
		for _, user := range users {
			if err := db.Delete(user).Error; err != nil {
				t.Fatal(err)
			}
			delete(userLocks.m, user.Id)
		}
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

	if err := LoadStocks(); err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		u, err := GetUserCopy(tc.userId)
		if err != nil {
			t.Fatalf("Error loading user data %+v", err)
		}
		originalCash := u.Cash

		tr, err := PerformMortgageTransaction(tc.userId, tc.stockId, tc.stockQuantity, tc.retrievePrice)

		if err != nil {
			if _, ok := err.(NotEnoughStocksError); !ok {
				t.Fatalf("Did not expect error. Got %+v", err)
			}
			continue
		}

		// append the transaction to the user's list of transactions
		transactions = append(transactions, tr)

		stockLeft, err := getSingleStockCount(&u, tc.stockId)
		if err != nil {
			t.Fatalf("Error loading current stock count %+v", err)
		}
		if stockLeft != tc.stockLeft {
			t.Fatalf("StockLeft mismatch. Got %d, want %d; tc: %+v", stockLeft, tc.stockLeft, tc)
		}

		u, err = GetUserCopy(tc.userId)
		if err != nil {
			t.Fatalf("Error loading user data %+v; tc %+v", err, tc)
		}

		if tc.stockQuantity > 0 {
			if originalCash-tc.cashGained != u.Cash {
				t.Fatalf("Cash didn't change as expected. Want new cash %d; Got %+v; tc: %+v", originalCash+tc.cashGained, u.Cash, tc)
			}
		} else {
			if originalCash+tc.cashGained != u.Cash {
				t.Fatalf("Cash didn't change as expected. Want new cash %d; Got %+v; tc: %+v", originalCash+tc.cashGained, u.Cash, tc)
			}
		}
	}
}

func TestGetTaxPercent(t *testing.T) {
	testutils.AssertEqual(t, 0, getTaxPercent(-50))
	testutils.AssertEqual(t, 0, getTaxPercent(0))
	testutils.AssertEqual(t, 2, getTaxPercent(50000))
	testutils.AssertEqual(t, 5, getTaxPercent(200000))
	testutils.AssertEqual(t, 9, getTaxPercent(750000))
	testutils.AssertEqual(t, 15, getTaxPercent(1500000))
	testutils.AssertEqual(t, 25, getTaxPercent(2500000))
}
