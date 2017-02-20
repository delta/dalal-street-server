package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"

	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

var (
	UnauthorizedError  = errors.New("Invalid credentials")
	NotRegisteredError = errors.New("Not registered on main site")
	InternalError      = errors.New("Internal server error")
)

var TotalUserCount uint32

// User models the User object.
type User struct {
	Id        uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	Email     string `gorm:"unique;not null" json:"email"`
	Name      string `gorm:"not null" json:"name"`
	Cash      uint32 `gorm:"not null" json:"cash"`
	Total     int32  `gorm:"not null" json:"total"`
	CreatedAt string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (u *User) ToProto() *models_proto.User {
	return &models_proto.User{
		Id:        u.Id,
		Email:     u.Email,
		Name:      u.Name,
		Cash:      u.Cash,
		Total:     u.Total,
		CreatedAt: u.CreatedAt,
	}
}

// pragyanUser is the structure returned by Pragyan API
type pragyanUser struct {
	Id   uint32 `json:"user_id"`
	Name string `json:"user_fullname"`
}

// User.TableName() is for letting Gorm know the correct table name.
func (User) TableName() string {
	return "Users"
}

// Login() is used to login an existing user or register a new user
// Registration happens provided Pragyan API verifies the credentials
// and the user doesn't exist in our database.
func Login(email, password string) (User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":         "Login",
		"param_email":    email,
		"param_password": password,
	})

	l.Infof("Attempting to login user")

	l.Debugf("Trying to call Pragyan API for logging in")

	pu, err := postLoginToPragyan(email, password)
	if err != nil {
		l.Debugf("Pragyan API call failed")
		return User{}, err
	}

	l.Debugf("Trying to get user from database. UserId: %d, Name: %s", pu.Id, pu.Name)

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return User{}, InternalError
	}
	defer db.Close()

	u := &User{}
	if result := db.First(u, pu.Id); result.Error != nil {
		if !result.RecordNotFound() {
			l.Errorf("Error in loading user info from database: '%s'", result.Error)
			return User{}, InternalError
		}

		l.Infof("User (%d, %s, %s) not found in database. Registering new user", pu.Id, email, pu.Name)

		u, err = createUser(pu, email)
		if err != nil {
			return User{}, InternalError
		}
	}

	l.Infof("Found user (%d, %s, %s). Logging him in.", u.Id, u.Email, u.Name)

	return *u, nil
}

// createUser() creates a user given his email and name.
func createUser(pu pragyanUser, email string) (*User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":      "createUser",
		"param_id":    pu.Id,
		"param_email": email,
		"param_name":  pu.Name,
	})

	l.Infof("Creating user")

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	u := &User{
		Id:        pu.Id,
		Email:     email,
		Name:      pu.Name,
		Cash:      STARTING_CASH,
		Total:     STARTING_CASH,
		CreatedAt: time.Now().String(),
	}

	err = db.Save(u).Error

	if err != nil {
		l.Errorf("Failed: %+v", err)
		return nil, err
	}

	//update total user count
	atomic.AddUint32(&TotalUserCount, 1)

	l.Infof("Created user")
	return u, nil
}

// postLoginToPragyan() is used to make a post request to pragyan and return
// a pragyanUser struct.
func postLoginToPragyan(email, password string) (pragyanUser, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":      "postLoginToPragyan",
		"param_email": email,
		"param_name":  password,
	})

	form := url.Values{
		"user_email":   {email},
		"user_pass":    {password},
		"event_id":     {utils.Configuration.EventId},
		"event_secret": {utils.Configuration.EventSecret},
	}

	l.Debugf("Attempting login to Pragyan")
	resp, err := http.PostForm("https://api.pragyan.org/event/login", form)
	if err != nil {
		l.Errorf("Pragyan API call failed: '%s'", err)
		return pragyanUser{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Errorf("Failed to read Pragyan API's response: '%s'", err)
		return pragyanUser{}, err
	}
	l.Debugf("Pragyan API call response: '%s'", string(body))

	type message_t struct {
		Id   string `json:"user_id"`
		Name string `json:"user_fullname"`
	}
	r := struct {
		StatusCode int       `json:"status_code"`
		Message    message_t `json:"message"`
	}{}
	json.Unmarshal(body, &r)

	switch r.StatusCode {
	case 200:
		uid, _ := strconv.ParseUint(r.Message.Id, 10, 32)
		pu := pragyanUser{
			Id:   uint32(uid),
			Name: r.Message.Name,
		}

		l.Debugf("Credentials verified. UserId: %d, Name: %s", pu.Id, pu.Name)

		return pu, nil
	case 401:
		l.Debugf("Bad credentials")
		return pragyanUser{}, UnauthorizedError
	case 412:
		l.Debugf("Not registered on main site")
		return pragyanUser{}, NotRegisteredError
	default:
		l.Errorf("Pragyan rejected API call with (%d, %s)", r.StatusCode, r.Message)
		return pragyanUser{}, InternalError
	}
}

//
// Private stuff
//

type userAndLock struct {
	sync.RWMutex
	user *User
}

// userLocks is used to synchronize access to a User's data.
// userLocks.m stores the locks for each user.
// Access to userLocks itself is synchronized via a RWMutex.
var userLocks = struct {
	sync.RWMutex
	m map[uint32]*userAndLock
}{m: make(map[uint32]*userAndLock)}

// getSingleStockCount() returns the stocks a user owns for a given stockId
// This method is *not* thread-safe.
// The caller is responsible for thread-safety.
func getSingleStockCount(u *User, stockId uint32) (int32, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "getSingleStockCount",
		"param_u":       fmt.Sprintf("%+v", u),
		"param_stockId": stockId,
	})

	l.Debugf("Attempting")

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return 0, err
	}
	defer db.Close()

	var stockCount = struct{ Sc int32 }{0}
	sql := "Select sum(StockQuantity) as sc from Transactions where UserId=? and StockId=?"
	if err := db.Raw(sql, u.Id, stockId).Scan(&stockCount).Error; err != nil {
		l.Error(err)
		return 0, err
	}

	l.Debugf("Got %d", stockCount.Sc)

	return stockCount.Sc, nil
}

//
// Public interface
//

//
// List of errors
//

// AskLimitExceededError is generated when an Ask's StockQuantity is greater
// than ASK_LIMIT
type AskLimitExceededError struct{}

func (e AskLimitExceededError) Error() string {
	return fmt.Sprintf("Ask orders must not exceed %d stocks per order", ASK_LIMIT)
}

// BuyLimitExceededError is generated when an Bid's StockQuantity is greater
// than BUY_LIMIT (for BuyFromExchange orders)
type BuyLimitExceededError struct{}

func (e BuyLimitExceededError) Error() string {
	return fmt.Sprintf("BuyFromExchange orders must not exceed %d stocks per order", BUY_LIMIT)
}

// NotEnoughCashError is generated when an Ask's StockQuantity is such that
// deducting those many stocks will leave the user with less than
// SHORT_SELL_BORROW_LIMIT
type NotEnoughStocksError struct {
	currentAllowedQty int32
}

func (e NotEnoughStocksError) Error() string {
	return fmt.Sprintf("Not have enough stocks to place this order. Current maximum Ask size: %d stocks", e.currentAllowedQty)
}

// BidLimitExceededError is generated when a Bid's StockQuantity is greater than BID_LIMIT
type BidLimitExceededError struct{}

func (e BidLimitExceededError) Error() string {
	return fmt.Sprintf("Bid orders must not exceed %d stocks per bid", BID_LIMIT)
}

// NotEnoughCashError is generated when a Bid's StockQuantity*Price is such that
// deducting so much cash from the user will leave him with less than
// MINIMUM_CASH_LIMIT
type NotEnoughCashError struct{}

func (e NotEnoughCashError) Error() string {
	return fmt.Sprintf("Not enough cash to place this order")
}

type InvalidAskIdError struct{}

func (e InvalidAskIdError) Error() string {
	return fmt.Sprintf("Invalid ask id")
}

type InvalidBidIdError struct{}

func (e InvalidBidIdError) Error() string {
	return fmt.Sprintf("Invalid bid id")
}

//
// Methods
//

// getUser() gets a user by his id.
// u method returns a channel and a pointer to the user object. The callee
// is guaranteed to get exclusive write access to the user object. Once the callee
// is done using the object, he must close the channel.
func getUser(id uint32) (chan struct{}, *User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "getUser",
		"param_id": id,
	})

	var (
		u  = &userAndLock{sync.RWMutex{}, &User{}}
		ch = make(chan struct{})
	)

	l.Debugf("Attempting")

	/* Try to see if the user is there in the map */
	userLocks.Lock()
	defer func() {
		userLocks.Unlock()
		l.Debugf("Unlocked userLocks map")
	}()

	l.Debugf("Locked userLocks map")

	_, ok := userLocks.m[id]
	if ok {
		u = userLocks.m[id]
		l.Debugf("Found user in userLocks map. Locking.")
		u.Lock()
		go func() {
			l.Debugf("Waiting for caller to release lock")
			<-ch
			u.Unlock()
			l.Debugf("Lock released")
		}()
		return ch, u.user, nil
	}

	/* Otherwise load from database */
	l.Debugf("Loading user from database")
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, nil, err
	}
	defer db.Close()

	userLocks.m[id] = u
	if errDb := db.First(u.user, id); errDb.Error != nil {
		if errDb.RecordNotFound() {
			l.Errorf("Attempted to get non-existing user")
			return nil, nil, fmt.Errorf("User with Id %d does not exist", id)
		} else {
			return nil, nil, errDb.Error
		}
	}

	l.Debugf("Loaded user from db. Locking.")
	u.Lock()
	go func() {
		l.Debugf("Waiting for caller to release lock")
		<-ch
		u.Unlock()
		l.Debugf("Lock released")
	}()

	l.Debugf("User: %+v", u.user)

	return ch, u.user, nil
}

// GetUserCopy() gets a copy of user by his id.
// The method returns a channel and a copy of the user object. The callee
// is guaranteed to get read access to the user object. Once the callee
// is done using the object, he must close the channel.
func GetUserCopy(id uint32) (User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "GetUserCopy",
		"param_id": id,
	})

	var u = &userAndLock{sync.RWMutex{}, &User{}}

	l.Debugf("Attempting")

	/* Try to see if the user is there in the map */
	userLocks.Lock()
	defer userLocks.Unlock()

	_, ok := userLocks.m[id]
	if ok {
		u = userLocks.m[id]
		l.Debugf("Found user in userLocks map. Locking.")
		u.RLock()
		defer u.RUnlock()
		return *u.user, nil
	}

	/* Otherwise load from database */
	l.Debugf("Loading user from database")
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return User{}, err
	}
	defer db.Close()

	userLocks.m[id] = u
	if errDb := db.First(u.user, id); errDb.Error != nil {
		if errDb.RecordNotFound() {
			l.Errorf("Attempted to get non-existing user")
			return User{}, fmt.Errorf("User with Id %d does not exist", id)
		} else {
			return User{}, errDb.Error
		}
	}

	l.Debugf("Loaded user from db. Locking.")
	u.RLock()
	defer u.RUnlock()
	l.Debugf("User: %+v", u.user)

	return *u.user, nil
}

// User.PlaceAskOrder() places an Ask order for the user.
//
// The method is thread-safe like other exported methods of this package.
//
// Possible outcomes:
// 	1. Ask gets placed successfully
//  2. AskLimitExceededError is returned
//  3. NotEnoughStocksError is returned
//  4. Other error is returned (e.g. if Database connection doesn't open)
func PlaceAskOrder(userId uint32, ask *Ask) (uint32, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":       "PlaceAskOrder",
		"param_userId": userId,
		"param_ask":    fmt.Sprintf("%+v", ask),
	})

	l.Infof("Attempting")

	l.Debugf("Acquiring exclusive write on user")

	ch, user, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return 0, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	// A lock on user has been acquired.
	// Safe to make changes to this user.

	// First check: Order size should be less than ASK_LIMIT
	l.Debugf("Check1: Order size vs ASK_LIMIT (%d)", ASK_LIMIT)
	if ask.StockQuantity > ASK_LIMIT {
		l.Debugf("Check1: Failed.")
		return 0, AskLimitExceededError{}
	}
	l.Debugf("Check1: Passed.")

	// Second Check: User should have enough stocks
	numStocks, err := getSingleStockCount(user, ask.StockId)
	if err != nil {
		return 0, err
	}
	var numStocksLeft = numStocks - int32(ask.StockQuantity)

	l.Debugf("Check2: Current stocks: %d. Stocks after trade: %d.", numStocks, numStocksLeft)

	if numStocksLeft < -SHORT_SELL_BORROW_LIMIT {
		l.Debugf("Check2: Failed. Not enough stocks.")
		currentAllowedQty := numStocks + SHORT_SELL_BORROW_LIMIT
		if currentAllowedQty > ASK_LIMIT {
			currentAllowedQty = ASK_LIMIT
		}
		return 0, NotEnoughStocksError{currentAllowedQty}
	}

	l.Debugf("Check2: Passed. Creating Ask.")

	if err := createAsk(ask); err != nil {
		l.Errorf("Error creating the ask %+v", err)
		return 0, err
	}

	l.Infof("Created Ask order. AskId: ", ask.Id)

	/*
		AddAskOrder(ask, PerformOrderFillTransacction)
	*/

	return ask.Id, nil
}

// User.PlaceBidOrder() places a Bid order for the user.
// The method is thread-safe like other exported methods of this package.
//
// Possible outcomes:
// 	1. Bid gets placed successfully
//  2. BidLimitExceededError is returned
//  3. NotEnoughCashError is returned
//  4. Other error is returned (e.g. if Database connection doesn't open)
func PlaceBidOrder(userId uint32, bid *Bid) (uint32, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":    "PlaceBidOrder",
		"param_id":  userId,
		"param_bid": fmt.Sprintf("%+v", bid),
	})

	l.Infof("PlaceBidOrder requested")

	l.Debugf("Acquiring exclusive write on user")
	ch, user, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return 0, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	// A lock on user has been acquired.
	// Safe to make changes to this user.

	// First check: Order size should be less than BID_LIMIT
	l.Debugf("Check1: Order size vs BID_LIMIT (%d)", BID_LIMIT)
	if bid.StockQuantity > BID_LIMIT {
		l.Debugf("Check1: Failed.")
		return 0, BidLimitExceededError{}
	}
	l.Debugf("Check1: Passed.")

	// Second Check: User should have enough cash
	var cashLeft = int32(user.Cash) - int32(bid.StockQuantity*bid.Price)

	l.Debugf("Check2: User has %d cash currently. Will be left with %d cash after trade.", user.Cash, cashLeft)

	if cashLeft < MINIMUM_CASH_LIMIT {
		l.Debugf("Check2: Failed. Not enough cash.")
		return 0, NotEnoughCashError{}
	}

	l.Debugf("Check2: Passed. Creating Bid")

	if err := createBid(bid); err != nil {
		l.Errorf("Error creating the bid %+v", err)
		return 0, err
	}

	l.Infof("Created Bid order. BidId: %d", bid.Id)

	return bid.Id, nil

}

func CancelOrder(userId uint32, orderId uint32, isAsk bool) error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "CancelOrder",
		"userId":  userId,
		"orderId": orderId,
		"isAsk":   isAsk,
	})

	l.Infof("CancelOrder requested")

	l.Debugf("Acquiring exclusive write on user")

	ch, _, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	if isAsk {
		l.Debugf("Acquiring lock on ask order")
		askOrder, err := getAsk(orderId)
		if askOrder == nil || askOrder.UserId != userId {
			l.Errorf("Invalid ask id provided")
			return InvalidAskIdError{}
		} else if err != nil {
			l.Errorf("Unknown error in getAsk: %+v", err)
			return err
		}

		if err := askOrder.Close(); err != nil {
			l.Errorf("Unknown error while saving that ask is cancelled %+v", err)
			return err
		}
	} else {
		l.Debugf("Acquiring lock on bid order")
		bidOrder, err := getBid(orderId)
		if bidOrder == nil || bidOrder.UserId != userId {
			l.Errorf("Invalid bid id provided")
			return InvalidBidIdError{}
		} else if err != nil {
			l.Errorf("Unknown error in getBid")
			return err
		}

		if err := bidOrder.Close(); err != nil {
			l.Errorf("Unknown error while saving that bid is cancelled %+v", err)
			return err
		}
	}

	l.Infof("Cancelled order")

	return nil
}

func PerformBuyFromExchangeTransaction(userId, stockId, stockQuantity uint32) (*Transaction, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "PerformBuyFromExchangeTransaction",
		"param_userId":        userId,
		"param_stockId":       stockId,
		"param_stockQuantity": stockQuantity,
	})

	l.Infof("PerformBuyFromExchangeTransaction requested")

	if stockQuantity > BUY_LIMIT {
		l.Debugf("Exceeded buy limit. PerformBuyFromExchange failing")
		return nil, BuyLimitExceededError{}
	}

	l.Debugf("Acquiring exclusive write on user")
	ch, user, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return nil, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	l.Debugf("Acquiring exclusive write on stock")

	allStocks.m[stockId].Lock()
	stock := allStocks.m[stockId].stock
	defer func() {
		l.Debugf("Released exclusive write on stock")
		allStocks.m[stockId].Unlock()
	}()

	// A lock on user and stock has been acquired.
	// Safe to make changes to this user and this stock

	stockQuantityRemoved := stockQuantity
	if stockQuantityRemoved > stock.StocksInExchange {
		stockQuantityRemoved = stock.StocksInExchange
	}

	if stockQuantityRemoved == 0 {
		return nil, NotEnoughStocksError{}
	}

	price := stock.CurrentPrice

	if price*stockQuantityRemoved > user.Cash {
		l.Debugf("User does not have enough cash. Want %d, Have %d. Failing.", price*stockQuantityRemoved, user.Cash)
		return nil, NotEnoughCashError{}
	}

	l.Debugf("%d stocks will be removed at %d per stock", stockQuantityRemoved, price)

	transaction := &Transaction{
		UserId:        userId,
		StockId:       stockId,
		Type:          FromExchangeTransaction,
		StockQuantity: int32(stockQuantityRemoved),
		Price:         price,
		Total:         -int32(price * stockQuantityRemoved),
	}

	userCash := uint32(int32(user.Cash) + transaction.Total)
	newStocksInExchange := stock.StocksInExchange - stockQuantityRemoved
	newStocksInMarket := stock.StocksInMarket + stockQuantityRemoved

	/* Committing to database */
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	tx := db.Begin()

	if err := tx.Save(transaction).Error; err != nil {
		l.Errorf("Error creating the transaction. Rolling back. Error: %+v", err)
		tx.Rollback()
		return nil, err
	}

	l.Debugf("Added transaction to Transactions table")

	if err := tx.Model(user).Update("cash", userCash).Error; err != nil {
		l.Errorf("Error deducting the cash from user's account. Rolling back. Error: %+v", err)
		tx.Rollback()
		return nil, err
	}

	l.Debugf("Deducted cash from user's account. New balance: %d", userCash)

	if err := tx.Model(stock).Updates(&Stock{StocksInExchange: newStocksInExchange, StocksInMarket: newStocksInMarket}).Error; err != nil {
		l.Errorf("Error transfering stocks from exchange to market")
		return nil, err
	}

	l.Debugf("Transferred stocks from Exchange to Market")

	if err := tx.Commit().Error; err != nil {
		l.Errorf("Error committing the transaction. Failing. %+v", err)
		return nil, err
	}

	stock.StocksInExchange = newStocksInExchange
	stock.StocksInMarket = newStocksInMarket
	user.Cash = userCash

	l.Infof("Committed transaction. Removed %d stocks @ %d per stock. Total cost = %d. New balance: %d", stockQuantityRemoved, price, price*stockQuantityRemoved, user.Cash)

	return transaction, nil
}

/*
*	NOTE: THIS FUNCTION ASSUMES BOTH THE USERS HAVE BEEN LOCKED BY THE CALLEE
*
*	PerformOrderFillTransaction performs the following function
*		- Check if bidder has enough cash. If not, return NotEnoughcashError
*		- Check if asker has enough stocks. If not, return NotEnoughStocksError
*		- Set transaction price based on order type(Limit, Market). StopLoss needs to be handled separately
*		- Calculate updated cash for biddingUser and askingUser
*		- Calculate StockQuantityFulfilled and IsClosed for askOrder and bidOrder
*		- Create database transaction. Transaction performs the following database manipulations
*				- save askTransaction, bidTransaction
*				- update biddingUser and askingUser cash
*				- update StockQuantityFulfilled and IsClosed for askOrder and bidOrder
*
*
*	Returns askDone, bidDone, Error
*		- if askDone is true, ask can be removed
*		- if bidDone is true, bid can be removed
 */
func PerformOrderFillTransaction(askingUser *User, biddingUser *User, ask *Ask, bid *Bid) (bool, bool, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "PerformOrderFillTransaction",
		"askingUserId":  ask.UserId,
		"biddingUserId": bid.UserId,
		"stockId":       ask.StockId,
	})

	l.Infof("PerformOrderFillTransaction requested for stock id %v", ask.StockId)

	/*
		if ask.isclosed() return askDone, bidNotDone
		if bid.isClosed() return askNotDone, bidDone

		stkTradeQty = min(ask.StQtyUnf.., bid.StkQtyUNfi)
		stkTradePrice = min(ask.Price, bid.price)
		if both marketorder:
			allstocks.m[stkid].RLock()
			stkTradePrice = allStocks.m[stkid].stock.CurrentPrice
			allstks.msdfsd.Runlock()

		condition1: bidder has enough money
			no: return (bidDone, askNotDone)

		condition2: asker has enough stocks
			no: return (bidNotDone, askDone)

		IN DB:
			transact( askingUser, -stkTradeQty, stkTradePrice, +stkTradeQty*stkTradePrice )
			transact( biddingUser, +stkTradeQty, stkTradePrice, -stkTradeQty*stkTradePrice )

			askingUser.Cash += tr1.total
			biddingUser.Cash += tr2.total

			asker.stkQtyFulfilled += stkTradeQty
			bidder.stkQtyFulfilled += stkTradeQty

			if askfulfilled:
				askdone = true
			if bidfulfilled:
				biddone = true

		if errored:
			return askNotDone, bidNotDone, error!


		updateStockPrice(stkid, stkTradePrice)

		return askdone,  biddone
	*/
	if ask.IsClosed {
		return true, false, nil
	}
	if bid.IsClosed {
		return false, true, nil
	}

	var bidUnfulfilledStockQuantity = bid.StockQuantity - bid.StockQuantityFulfilled
	var askUnfulfilledStockQuantity = ask.StockQuantity - ask.StockQuantityFulfilled

	stockTradeQty := int32(min(askUnfulfilledStockQuantity, bidUnfulfilledStockQuantity))
	var stockTradePrice uint32

	//set transaction price based on order type
	if ask.OrderType == Market && bid.OrderType == Market {
		allStocks.m[ask.StockId].RLock()
		stock, _ := allStocks.m[ask.StockId]
		stockTradePrice = stock.stock.CurrentPrice
		allStocks.m[ask.StockId].RUnlock()
	} else if ask.OrderType == Market {
		stockTradePrice = bid.Price
	} else if bid.OrderType == Market {
		stockTradePrice = ask.Price
	} else {
		stockTradePrice = min(ask.Price, bid.Price)
	}

	//Check if bidder has enough cash
	var cashLeft = int32(biddingUser.Cash) - int32(stockTradePrice)*stockTradeQty

	if cashLeft < MINIMUM_CASH_LIMIT {
		l.Debugf("Check1: Failed. Not enough cash.")
		return false, true, nil
	}

	//Check if askingUser has enough stocks
	numStocks, err := getSingleStockCount(askingUser, ask.StockId)
	if err != nil {
		return false, false, err
	}

	var numStocksLeft = numStocks - int32(stockTradeQty)

	if numStocksLeft < -SHORT_SELL_BORROW_LIMIT {
		l.Debugf("Check2: Failed. Not enough stocks.")
		currentAllowedQty := numStocks + SHORT_SELL_BORROW_LIMIT
		if currentAllowedQty > ASK_LIMIT {
			currentAllowedQty = ASK_LIMIT
		}
		return true, false, nil
	}

	//helper function to return a transaction object
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

	total := int32(stockTradePrice) * stockTradeQty

	askTransaction := makeTrans(ask.UserId, ask.StockId, OrderFillTransaction, -stockTradeQty, stockTradePrice, total)
	bidTransaction := makeTrans(bid.UserId, bid.StockId, OrderFillTransaction, stockTradeQty, stockTradePrice, -total)

	//calculate user's updated cash
	askingUserCash := askingUser.Cash + uint32(stockTradeQty)*stockTradePrice
	biddingUserCash := biddingUser.Cash - uint32(stockTradeQty)*stockTradePrice

	//calculate StockQuantityFulfilled and IsClosed for ask and bid order
	askStockQuantityFulfilled := ask.StockQuantityFulfilled + uint32(stockTradeQty)
	bidStockQuantityFulfilled := bid.StockQuantityFulfilled - uint32(stockTradeQty)

	var askIsClosed bool
	var bidIsClosed bool
	if ask.StockQuantity == askStockQuantityFulfilled {
		askIsClosed = true
	}
	if bid.StockQuantity == bidStockQuantityFulfilled {
		bidIsClosed = true
	}

	//Committing to database
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return false, false, err
	}
	defer db.Close()

	//Begin transaction
	tx := db.Begin()

	//save askTransaction
	if err := tx.Save(askTransaction).Error; err != nil {
		l.Errorf("Error creating the askTransaction. Rolling back. Error: %+v", err)
		tx.Rollback()
		return false, false, err
	}

	l.Debugf("Added askTransaction to Transactions table")

	//save bidTransaction
	if err := tx.Save(bidTransaction).Error; err != nil {
		l.Errorf("Error creating the bidTransaction. Rolling back. Error: %+v", err)
		tx.Rollback()
		return false, false, err
	}

	l.Debugf("Added bidTransaction to Transactions table")

	//update askingUser
	if err := tx.Model(askingUser).Update("cash", askingUserCash).Error; err != nil {
		l.Errorf("Error updating askingUser.Cash Rolling back. Error: %+v", err)
		tx.Rollback()
		return false, false, err
	}

	//update biddingUserCash
	if err := tx.Model(biddingUser).Update("cash", biddingUserCash).Error; err != nil {
		l.Errorf("Error updating biddingUser.Cash Rolling back. Error: %+v", err)
		tx.Rollback()
		return false, false, err
	}

	//update StockQuantityFulfilled and IsClosed for ask order
	if err := tx.Model(ask).Updates(Ask{StockQuantityFulfilled: askStockQuantityFulfilled, IsClosed: askIsClosed}).Error; err != nil {
		l.Errorf("Error updating ask.{StockQuantityFulfilled,IsClosed}. Rolling back. Error: %+v", err)
		tx.Rollback()
		return false, false, err
	}

	//update StockQuantityFulfilled and IsClosed for bid order
	if err := tx.Model(bid).Updates(Bid{StockQuantityFulfilled: bidStockQuantityFulfilled, IsClosed: bidIsClosed}).Error; err != nil {
		l.Errorf("Error updating bid.{StockQuantityFulfilled,IsClosed}. Rolling back. Error: %+v", err)
		tx.Rollback()
		return false, false, err
	}

	//Commit transaction
	if err := tx.Commit().Error; err != nil {
		l.Errorf("Error committing the transaction. Failing. %+v", err)
		return false, false, err
	}

	askingUser.Cash = askingUserCash
	biddingUser.Cash = biddingUserCash
	ask.IsClosed = askIsClosed
	ask.StockQuantityFulfilled = askStockQuantityFulfilled
	bid.IsClosed = bidIsClosed
	bid.StockQuantityFulfilled = bidStockQuantityFulfilled

	l.Infof("Transaction committed successfully. Traded %d at %d per stock. Total %d.", stockTradeQty, stockTradePrice, total)

	if err := UpdateStockPrice(ask.StockId, stockTradePrice); err != nil {
		l.Errorf("Error updating stock price. BUT SUPRRESSING IT.")
		return false, false, nil // supress this error!
	}

	return ask.IsClosed, bid.IsClosed, nil
}

func PerformMortgageTransaction(userId, stockId uint32, stockQuantity int32) (*Transaction, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "PerformMortgageTransaction",
		"param_userId":        userId,
		"param_stockId":       stockId,
		"param_stockQuantity": stockQuantity,
	})

	l.Infof("PerformMortgageTransaction requested")

	l.Debugf("Acquiring exclusive write on user")
	ch, user, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return nil, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	allStocks.m[stockId].RLock()
	currentStockPrice := allStocks.m[stockId].stock.CurrentPrice
	allStocks.m[stockId].RUnlock()

	l.Debugf("Taking current price of stock as %d", currentStockPrice)

	var rate int32

	if stockQuantity >= 0 {
		rate = MORTGAGE_RETRIEVE_RATE
		l.Debugf("stockQuantity positive. Retrieving stocks from mortgage @ %d%%", rate)

		db, err := DbOpen()
		if err != nil {
			l.Error(err)
			return nil, err
		}
		defer db.Close()

		stockCount := struct{ Sc int32 }{0}
		sql := "Select sum(StockQuantity) as sc from Transactions where UserId=? and StockId=? and Type=?"
		err = db.Raw(sql, user.Id, stockId, MortgageTransaction.String()).Scan(&stockCount).Error
		if err != nil {
			l.Error(err)
			return nil, err
		}

		// Sc will be negative if stocks are there in mortgage.
		// Change sign to mean number of stocks in mortgage
		stockCount.Sc *= -1

		l.Debugf("%d stocks mortgaged currently", stockCount.Sc)

		if stockQuantity > stockCount.Sc {
			l.Errorf("Insufficient stocks in mortgage. Have %d, want %d", stockCount.Sc, stockQuantity)
			return nil, NotEnoughStocksError{}
		}

	} else {
		rate = MORTGAGE_DEPOSIT_RATE
		l.Debugf("stockQuantity negative. Depositing stocks to mortgage @ %d%%", rate)

		stockOwned, err := getSingleStockCount(user, stockId)
		if err != nil {
			l.Error(err)
			return nil, err
		}

		if stockOwned < -stockQuantity {
			l.Errorf("Insufficient stocks in ownership. Have %d, want %d", stockOwned, stockQuantity)
			return nil, NotEnoughStocksError{}
		}
	}

	trTotal := -int32(currentStockPrice) * stockQuantity * rate / 100
	if int32(user.Cash)+trTotal < 0 {
		l.Debugf("User does not have enough cash. Want %d, Have %d. Failing.", trTotal, user.Cash)
		return nil, NotEnoughCashError{}
	}

	transaction := &Transaction{
		UserId:        userId,
		StockId:       stockId,
		Type:          MortgageTransaction,
		StockQuantity: stockQuantity,
		Price:         0,
		Total:         trTotal,
	}

	// A lock on user and stock has been acquired.
	// Safe to make changes to this user and this stock

	user.Cash = uint32(int32(user.Cash) + trTotal)

	/* Committing to database */
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	tx := db.Begin()

	if err := tx.Save(transaction).Error; err != nil {
		l.Errorf("Error creating the transaction. Rolling back. Error: %+v", err)
		tx.Rollback()
		return nil, err
	}

	l.Debugf("Added transaction to Transactions table")

	if err := tx.Save(user).Error; err != nil {
		l.Errorf("Error updating user's cash. Rolling back. Error: %+v", err)
		tx.Rollback()
		return nil, err
	}

	l.Debugf("Updated user's cash. New balance: %d", user.Cash)

	if err := tx.Commit().Error; err != nil {
		l.Errorf("Error committing the transaction. Failing. %+v", err)
		return nil, err
	}

	l.Debugf("Committed transaction. Success.")

	return transaction, nil

}

/*
func PerformDividendTransaction(stockId, dividendPercent uint32) (err error) {
	var l = logger.WithFields(logrus.Fields{
		"method":    "PerformDividendTransaction",
		"param_stockId": stockId,
		"param_dividendPercent": dividendPercent,
	})

	l.Info("PerformDividendTransaction requested")

	l.Debug("Acquiring exclusive write on user")
	ch, user, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return err
	}
	l.Debug("Acquired")
	defer func() {
		close(ch)
		l.Debug("Released exclusive write on user")
	}()

	// A lock on user has been acquired.
	// Safe to make changes to this user

	stockQuantityOwned, err := getSingleStockCount(user, stockId)
	price = stock.CurrentPrice

	if price * stockQuantityRemoved > user.Cash {
		l.Debugf("User does not have enough cash. Want %d, Have %d. Failing.", price * stockQuantityRemoved, user.Cash)
		return NotEnoughCashError{}, 0, 0
	}

	l.Debugf("%d stocks will be removed at %d per stock", stockQuantityRemoved, price)

	transaction := &Transaction{
		UserId: userId,
		StockId: stockId,
		Type: FromExchangeTransaction,
		StockQuantity: int32(stockQuantityRemoved),
		Price: price,
		Total: -int32(price * stockQuantityRemoved),
	}

	user.Cash = uint32(int32(user.Cash) + transaction.Total)

	/* Committing to database * /
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err, 0, 0
	}
	defer db.Close()

	tx := db.Begin()

	if err := tx.Save(transaction).Error; err != nil {
		l.Errorf("Error creating the transaction. Rolling back. Error: %+v", err)
		tx.Rollback()
		return err, 0, 0
	}

	l.Debugf("Added transaction to Transactions table")

	if err := tx.Save(user).Error; err != nil {
		l.Errorf("Error deducting the cash from user's account. Rolling back. Error: %+v", err)
		tx.Rollback()
		return err, 0, 0
	}

	l.Debugf("Deducted cash from user's account. New balance: %d", user.Cash)

	if err := tx.Commit().Error; err != nil {
		l.Errorf("Error committing the transaction. Failing. %+v", err)
		return err, 0, 0
	}

	l.Debugf("Committed transaction. Success.")

	return nil, stockQuantityRemoved, price
}
*/

type StockOwned struct {
	StockId       uint32
	StockQuantity int32
}

func GetStocksOwned(userId uint32) (map[uint32]int32, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetStocksOwned",
		"userId": userId,
	})

	l.Info("GetStocksOwned requested")

	l.Debugf("Acquiring lock on user")

	ch, _, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return nil, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released lock on user")
	}()

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, InternalError
	}
	defer db.Close()

	sql := "Select stockId, sum(stockQuantity) as stockQuantity from Transactions where userId=? group by stockId"
	rows, err := db.Raw(sql, userId).Rows()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer rows.Close()

	stocksOwned := make(map[uint32]int32)
	for rows.Next() {
		var stockId uint32
		var stockQty int32
		rows.Scan(&stockId, &stockQty)

		stocksOwned[stockId] = stockQty
	}

	return stocksOwned, nil
}

// Call User.Unload() when a user logs out. This will remove him from RAM
func (u *User) Unload() {

}
