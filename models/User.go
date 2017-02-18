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
		l.Debug("Bad credentials")
		return pragyanUser{}, UnauthorizedError
	case 412:
		l.Debug("Not registered on main site")
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
	defer userLocks.Unlock()

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

	l.Debug("Attempting")

	/* Try to see if the user is there in the map */
	userLocks.Lock()
	defer userLocks.Unlock()

	u, ok := userLocks.m[id]
	if ok {
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
		l.Debug("Released exclusive write on user")
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

	l.Info("Created Ask order. AskId: ", ask.Id)

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

	l.Info("PlaceBidOrder requested")

	l.Debug("Acquiring exclusive write on user")
	ch, user, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return 0, err
	}
	l.Debug("Acquired")
	defer func() {
		close(ch)
		l.Debug("Released exclusive write on user")
	}()

	// A lock on user has been acquired.
	// Safe to make changes to this user.

	// First check: Order size should be less than BID_LIMIT
	l.Debug("Check1: Order size vs BID_LIMIT (%d)", BID_LIMIT)
	if bid.StockQuantity > BID_LIMIT {
		l.Debug("Check1: Failed.")
		return 0, BidLimitExceededError{}
	}
	l.Debug("Check1: Passed.")

	// Second Check: User should have enough cash
	var cashLeft = int32(user.Cash) - int32(bid.StockQuantity*bid.Price)

	l.Debugf("Check2: User has %d cash currently. Will be left with %d cash after trade.", user.Cash, cashLeft)

	if cashLeft < MINIMUM_CASH_LIMIT {
		l.Debug("Check2: Failed. Not enough cash.")
		return 0, NotEnoughStocksError{}
	}

	l.Debug("Check2: Passed. Creating Bid")

	if err := createBid(bid); err != nil {
		l.Errorf("Error creating the bid %+v", err)
		return 0, err
	}

	l.Info("Created Bid order. BidId: %d", bid.Id)

	return bid.Id, nil

}

func CancelOrder(userId uint32, orderId uint32, isAsk bool) error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "CancelOrder",
		"userId":  userId,
		"orderId": orderId,
		"isAsk":   isAsk,
	})

	l.Info("CancelOrder requested")

	l.Debugf("Acquiring exclusive write on user")

	ch, _, err := getUser(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debug("Released exclusive write on user")
	}()

	if isAsk {
		l.Debug("Acquiring lock on ask order")
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
		l.Debug("Acquiring lock on bid order")
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

	l.Info("Cancelled order")

	return nil
}

func (u *User) PerformBuyFromExchangeTransaction() {

}

func (u *User) PerformOrderFillTransaction() {

}

func (u *User) PerformMortgageTransaction() {

}

func (u *User) PerformDividendTransaction() {

}

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
		l.Debug("Released lock on user")
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
