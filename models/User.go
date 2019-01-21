package models

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/delta/dalal-street-server/utils"
	"github.com/jinzhu/gorm"

	"github.com/Sirupsen/logrus"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
)

var (
	UnauthorizedError      = errors.New("Invalid credentials")
	NotRegisteredError     = errors.New("Not registered")
	AlreadyRegisteredError = errors.New("Already registered")
)

var TotalUserCount uint32

// User models the User object.
type User struct {
	Id        uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	Email     string `gorm:"unique;not null" json:"email"`
	Name      string `gorm:"not null" json:"name"`
	Cash      uint64 `gorm:"not null" json:"cash"`
	Total     int64  `gorm:"not null" json:"total"`
	CreatedAt string `gorm:"column:createdAt;not null" json:"created_at"`
	IsHuman   bool   `gorm:"column:isHuman;not null" json:"is_human"`
}

func (u *User) ToProto() *models_pb.User {
	return &models_pb.User{
		Id:        u.Id,
		Email:     u.Email,
		Name:      u.Name,
		Cash:      u.Cash,
		Total:     u.Total,
		CreatedAt: u.CreatedAt,
		IsHuman:   u.IsHuman,
	}
}

// pragyanUser is the structure returned by Pragyan API
type pragyanUser struct {
	Id      uint32 `json:"user_id"`
	Name    string `json:"user_fullname"`
	Country string `json:"user_country"`
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

	db := getDB()

	var registeredUser = Registration{
		Email: email,
	}

	err := db.Where("email = ?", email).First(&registeredUser).Error
	l.Debugf("Got %v while searching for registeredUser", err)

	// User was not found in our local db
	// Thus, he's logging in with pragyan for the first time
	if err != nil {
		//So register him if pragyan returns 200
		l.Debugf("Trying to call Pragyan API for logging in")

		pu, err := postLoginToPragyan(email, password)

		if err != nil {
			return User{}, err
		}

		// Pragyan returned 200
		// Add entry to Users table
		u, err := createUser(pu.Name, email)
		if err != nil {
			return User{}, err
		}
		register := &Registration{
			Email:      email,
			IsPragyan:  true,
			IsVerified: true,
			Name:       pu.Name,
			UserId:     u.Id,
		}

		// Add entry to Registrations table
		err = db.Save(register).Error
		if err != nil {
			// If registration failed, remove user from Users table as well
			db.Delete(u)
			return User{}, err
		}
		return *u, nil
	}

	getUserFromDB := func(userId uint32) (User, error) {
		l.Debugf("Trying to get user from database. UserId: %d", userId)
		u := User{}
		if result := db.First(&u, userId); result.Error != nil {
			if !result.RecordNotFound() {
				l.Errorf("Error in loading user info from database: '%s'", result.Error)
				return User{}, result.Error
			}
			// This case shouldn't happen!
			l.Warnf("User (%d, %s) not found in database", userId, email)
		}
		return u, nil
	}

	// Found in our local database
	// If he's not registered with Pragyan, match password
	if registeredUser.IsPragyan == false {
		if checkPasswordHash(password, registeredUser.Password) {
			return getUserFromDB(registeredUser.UserId)
		}
		return User{}, UnauthorizedError
	}
	//Registered with pragyan so hit pragyan with the username and password
	_, err = postLoginToPragyan(email, password)

	if err != nil {
		switch err {
		case UnauthorizedError:
			// Pragyan returned unauthorized
			// Thus, wrong password but user is registered
			return User{}, UnauthorizedError
		case NotRegisteredError:
			// Should never happen but is handled just in case
			// User once registered with pragyan creds but has now been deleted from the pragyan db
			db.Delete(&User{Id: registeredUser.Id})
			db.Delete(&registeredUser)
			return User{}, NotRegisteredError
		default:
			return User{}, err
		}
	}
	// Pragyan returned 200 hence use our db's user Id to load User
	return getUserFromDB(registeredUser.UserId)
}

// RegisterUser is called when a user tries to sign up in our site
func RegisterUser(email, password, fullName string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":         "Register",
		"param_email":    email,
		"param_password": password,
	})
	l.Debugf("Attempting to register user")

	db := getDB()

	var registeredUser = Registration{
		Email: email,
	}

	err := db.Where("email = ?", email).First(&registeredUser).Error

	if err == nil {
		return AlreadyRegisteredError
	}
	l.Debugf("Trying to call Pragyan API for checking if email available with Pragyan")
	_, err = postLoginToPragyan(email, password)

	if err == UnauthorizedError || err == nil {
		l.Error("User registered with pragyan ask to login with Pragyan")
		return AlreadyRegisteredError
	}
	if err != NotRegisteredError {
		l.Errorf("Unexpected error: %+v", err)
		return err
	}
	u, err := createUser(fullName, email)
	if err != nil {
		l.Errorf("Server error in Create user while logging in Pragyan user for the first time: %+v", err)
		return err
	}
	password, _ = hashPassword(password)
	register := &Registration{
		Email:      email,
		Password:   password,
		IsPragyan:  false,
		IsVerified: false,
		Name:       fullName,
		UserId:     u.Id,
	}
	err = db.Save(register).Error
	if err != nil {
		return err
	}

	return nil
}

func hashPassword(password string) (string, error) {
	h := sha1.New()
	h.Write([]byte(password))
	sha1Hash := hex.EncodeToString(h.Sum(nil))
	return sha1Hash, nil
}

func checkPasswordHash(password, hash string) bool {
	h := sha1.New()
	h.Write([]byte(password))
	sha1Hash := hex.EncodeToString(h.Sum(nil))
	return hash == sha1Hash
}

func getOrderFeePrice(price uint64, stockId uint32, o OrderType) uint64 {
	var orderFeePrice uint64
	switch o {
	case Limit:
		orderFeePrice = price
	case Market:
		allStocks.m[stockId].RLock()
		orderFeePrice = allStocks.m[stockId].stock.CurrentPrice
		allStocks.m[stockId].RUnlock()
	case StopLoss:
		orderFeePrice = price
	}
	return orderFeePrice
}

func getOrderFee(quantity, price uint64) uint64 {
	return uint64((ORDER_FEE_PERCENT / 100.0) * float64(quantity*price))
}

// createUser() creates a user given his email and name.
func createUser(name string, email string) (*User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":      "createUser",
		"param_email": email,
		"param_name":  name,
	})

	l.Infof("Creating user")

	db := getDB()

	u := &User{
		Email:     email,
		Name:      name,
		Cash:      STARTING_CASH,
		Total:     STARTING_CASH,
		CreatedAt: utils.GetCurrentTimeISO8601(),
		IsHuman:   true,
	}

	err := db.Save(u).Error
	if err != nil {
		l.Errorf("Failed: %+v", err)
		return nil, err
	}

	//update total user count
	atomic.AddUint32(&TotalUserCount, 1)

	l.Infof("Created user")
	return u, nil
}

// CreateBot creates a bot from the botName
func CreateBot(botName string) (*User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "createUser",
		"bot_name": botName,
	})

	l.Infof("Creating user")

	db := getDB()

	u := &User{
		Email:     botName + "@bot",
		Name:      botName,
		Cash:      STARTING_CASH,
		Total:     STARTING_CASH,
		CreatedAt: utils.GetCurrentTimeISO8601(),
	}

	err := db.Save(u).Error

	if err != nil {
		l.Errorf("Failed: %+v", err)
		return nil, err
	}

	//update total user count
	atomic.AddUint32(&TotalUserCount, 1)

	l.Infof("Created Bot")
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
		"event_id":     {config.EventId},
		"event_secret": {config.EventSecret},
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

	r := struct {
		StatusCode int         `json:"status_code"`
		Message    interface{} `json:"message"` // sometimes a pragyanUser, sometimes a string. :(
	}{}
	json.Unmarshal(body, &r)

	switch r.StatusCode {
	case 200:
		pu := pragyanUser{}
		userInfoMap := r.Message.(map[string]interface{})
		pu.Id = uint32(userInfoMap["user_id"].(float64)) // sigh. Have to do this because Message is interface{}
		pu.Name = userInfoMap["user_fullname"].(string)
		switch country := userInfoMap["user_country"].(type) {
		case string:
			pu.Country = country
		case nil:
			pu.Country = "India"
		}

		l.Debugf("Credentials verified. UserId: %d, Name: %s", pu.Id, pu.Name)

		return pu, nil
	case 401:
		l.Debugf("Bad credentials")
		return pragyanUser{}, UnauthorizedError
	case 412: // Not sure if this is needed. Unaware of the current API changes
		l.Debugf("Not registered on main site")
		return pragyanUser{}, NotRegisteredError
	case 400:
		if r.Message == "Account Not Registered" {
			return pragyanUser{}, NotRegisteredError
		}
		fallthrough
	default:
		l.Errorf("Pragyan rejected API call with (%d, %s)", r.StatusCode, r.Message)
		return pragyanUser{}, fmt.Errorf("Unexpected response from Pragyan: %+v", r)
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
func getSingleStockCount(u *User, stockId uint32) (int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "getSingleStockCount",
		"param_u":       fmt.Sprintf("%+v", u),
		"param_stockId": stockId,
	})

	l.Debugf("Attempting")

	db := getDB()

	var stockCount = struct{ Sc int64 }{0}
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

// OrderStockLimitExceeded is generated when an order's StockQuantity is greater than ASK/BID_LIMIT
type OrderStockLimitExceeded struct{}

func (e OrderStockLimitExceeded) Error() string {
	return fmt.Sprintf("An order can involve a trade of min 1 and max %d stocks", ASK_LIMIT)
}

// OrderPriceOutOfWindowError is sent when the order's price is outside the allowed price window
type OrderPriceOutOfWindowError struct{ price uint64 }

func (e OrderPriceOutOfWindowError) Error() string {
	return fmt.Sprintf("Order price must be within %d%% of the current price - currently between %d and %d",
		ORDER_PRICE_WINDOW,
		uint64((1-ORDER_PRICE_WINDOW/100.0)*float64(e.price)),
		uint64((1+ORDER_PRICE_WINDOW/100.0)*float64(e.price)),
	)
}

// MinimumPriceThresholdError is sent when the order's price is below MINIMUM_ORDER_PRICE
type MinimumPriceThresholdError struct{}

func (e MinimumPriceThresholdError) Error() string {
	return fmt.Sprintf("Order price must be above %d", MINIMUM_ORDER_PRICE)
}

// BuyLimitExceededError is generated when an Bid's StockQuantity is greater
// than BUY_LIMIT (for BuyFromExchange orders)
type BuyLimitExceededError struct{}

func (e BuyLimitExceededError) Error() string {
	return fmt.Sprintf("BuyFromExchange orders must not exceed %d stocks per order", BUY_LIMIT)
}

// NotEnoughStocksError is generated when an Ask's StockQuantity is such that
// deducting those many stocks will leave the user with less than
// SHORT_SELL_BORROW_LIMIT
type NotEnoughStocksError struct {
	currentAllowedQty int64
}

func (e NotEnoughStocksError) Error() string {
	return fmt.Sprintf("Not have enough stocks to place this order. Current maximum Ask size: %d stocks", e.currentAllowedQty)
}

// NotEnoughCashError is generated when a Bid's StockQuantity*Price is such that
// deducting so much cash from the user will leave him with less than
// MINIMUM_CASH_LIMIT
type NotEnoughCashError struct{}

func (e NotEnoughCashError) Error() string {
	return fmt.Sprintf("Not enough cash to place this order")
}

// InvalidOrderIDError is given out when a user tries to cancel an order he didn't make or that didn't exist
type InvalidOrderIDError struct{}

func (e InvalidOrderIDError) Error() string {
	return fmt.Sprintf("Invalid order id")
}

// InvalidRetrievePriceError is given out when a user tries to cancel an order he didn't make or that didn't exist
type InvalidRetrievePriceError struct{}

func (e InvalidRetrievePriceError) Error() string {
	return fmt.Sprintf("Invalid retrieve price")
}

//
// Methods
//

// getUserExclusively() gets a user by his id.
// u method returns a channel and a pointer to the user object. The callee
// is guaranteed to get exclusive write access to the user object. Once the callee
// is done using the object, he must close the channel.
func getUserExclusively(id uint32) (chan struct{}, *User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "getUserExclusively",
		"param_id": id,
	})

	var (
		u  = &userAndLock{sync.RWMutex{}, &User{}}
		ch = make(chan struct{})
	)

	l.Debugf("Attempting")

	/* Try to see if the user is there in the map */
	userLocks.Lock()
	l.Debugf("Locked userLocks map")

	_, ok := userLocks.m[id]
	if ok {
		u = userLocks.m[id]
		userLocks.Unlock()
		l.Debugf("Found user in userLocks map. Unlocked the map")
		u.Lock()
		l.Debugf("Locked user")
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
	db := getDB()

	userLocks.m[id] = u
	db = db.First(u.user, id)
	userLocks.Unlock()
	l.Debugf("Talked to db. Unlocked userLocks.")

	if db.Error != nil {
		if db.RecordNotFound() {
			l.Errorf("Attempted to get non-existing user")
			return nil, nil, fmt.Errorf("User with Id %d does not exist", id)
		}
		return nil, nil, db.Error
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

func getUserPairExclusive(id1, id2 uint32) (chan struct{}, *User, *User, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":    "getUserExclusively",
		"param_id1": id1,
		"param_id2": id2,
	})

	l.Debugf("Acquiring lock in order of User Ids")

	var firstUserId, secondUserId uint32

	//look out for error!!!
	if id1 < id2 {
		firstUserId = id1
		secondUserId = id2
	} else {
		firstUserId = id2
		secondUserId = id1
	}

	l.Debugf("Want first and second as %d, %d", firstUserId, secondUserId)
	defer l.Debugf("Closed channels of %d and %d", firstUserId, secondUserId)

	firstLockChan, firstUser, err := getUserExclusively(firstUserId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return nil, nil, nil, err
	}

	secondLockChan, secondUser, err := getUserExclusively(secondUserId)
	if err != nil {
		close(firstLockChan)
		l.Errorf("Errored: %+v", err)
		return nil, nil, nil, err
	}

	l.Debugf("Acquired the locks on users %d and %d", firstUserId, secondUserId)

	done := make(chan struct{})
	go func() {
		<-done
		close(secondLockChan)
		close(firstLockChan)
	}()

	if firstUserId == id1 {
		return done, firstUser, secondUser, nil
	}
	return done, secondUser, firstUser, nil
}

// GetUserCopy gets a copy of user by his id.
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
	l.Debugf("Locked userLocks map")

	_, ok := userLocks.m[id]
	if ok {
		u = userLocks.m[id]
		userLocks.Unlock()
		l.Debugf("Found user in userLocks map. Unlocked map.")
		u.RLock()
		defer u.RUnlock()
		return *u.user, nil
	}

	/* Otherwise load from database */
	l.Debugf("Loading user from database")
	db := getDB()

	userLocks.m[id] = u
	db = db.First(u.user, id)
	l.Debugf("Talked to Db. Unlocking userLocks")
	userLocks.Unlock()

	if errDb := db.First(u.user, id); errDb.Error != nil {
		if errDb.RecordNotFound() {
			l.Errorf("Attempted to get non-existing user")
			return User{}, fmt.Errorf("User with Id %d does not exist", id)
		}
		return User{}, errDb.Error
	}

	l.Debugf("Loaded user from db. Locking.")
	u.RLock()
	defer u.RUnlock()
	l.Debugf("User: %+v", u.user)

	return *u.user, nil
}

// SubtractOrderFee subtracts cash for users account
func SubtractOrderFee(user *User, orderFee uint64, tx *gorm.DB) error {
	l := logger.WithFields(logrus.Fields{
		"method":         "SubtractOrderFee",
		"param_user":     fmt.Sprintf("%+v", user),
		"param_orderFee": fmt.Sprintf("%d", orderFee),
	})

	user.Cash = user.Cash - orderFee

	if err := tx.Save(user).Error; err != nil {
		return err
	}

	l.Infof("Updated user cash. User now has %d", user.Cash)

	return nil
}

// PlaceAskOrder places an Ask order for the user.
//
// The method is thread-safe like other exported methods of this package.
//
// Possible outcomes:
// 	1. Ask gets placed successfully
//  2. AskLimitExceededError is returned
//  3. NotEnoughStocksError is returned
//  4. Other error is returned (e.g. if Database connection doesn't open)
func PlaceAskOrder(userId uint32, ask *Ask) (uint32, error) {
	l := logger.WithFields(logrus.Fields{
		"method":       "PlaceAskOrder",
		"param_userId": userId,
		"param_ask":    fmt.Sprintf("%+v", ask),
	})

	l.Infof("Attempting")

	// Place cap on order price only for limit orders
	if ask.OrderType == Limit {
		if ask.Price <= MINIMUM_ORDER_PRICE {
			l.Debugf("Minimum price check failed for ask order")
			return 0, MinimumPriceThresholdError{}
		}

		l.Debugf("Acquiring lock for ask order threshold check with stock id : %v ", ask.StockId)

		allStocks.m[ask.StockId].RLock()
		currentPrice := allStocks.m[ask.StockId].stock.CurrentPrice
		allStocks.m[ask.StockId].RUnlock()

		l.Debugf("Releasing lock for ask order threshold check with stock id : %v ", ask.StockId)

		var upperLimit = uint64((1 + ORDER_PRICE_WINDOW/100.0) * float64(currentPrice))
		var lowerLimit = uint64((1 - ORDER_PRICE_WINDOW/100.0) * float64(currentPrice))

		if ask.Price > upperLimit || ask.Price < lowerLimit {
			l.Debugf("Threshold price check failed for ask order")
			return 0, OrderPriceOutOfWindowError{currentPrice}
		}
	}

	l.Debugf("Acquiring exclusive write on user")

	ch, user, err := getUserExclusively(userId)
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
	if ask.StockQuantity > ASK_LIMIT || ask.StockQuantity < 1 {
		l.Debugf("Check1: Failed.")
		return 0, OrderStockLimitExceeded{}
	}
	l.Debugf("Check1: Passed.")

	// Second Check: User should have enough stocks
	numStocks, err := getSingleStockCount(user, ask.StockId)
	if err != nil {
		return 0, err
	}
	var numStocksLeft = numStocks - int64(ask.StockQuantity)

	l.Debugf("Check2: Current stocks: %d. Stocks after trade: %d.", numStocks, numStocksLeft)

	if numStocksLeft < -SHORT_SELL_BORROW_LIMIT {
		l.Debugf("Check2: Failed. Not enough stocks.")
		currentAllowedQty := numStocks + SHORT_SELL_BORROW_LIMIT
		if currentAllowedQty > ASK_LIMIT {
			currentAllowedQty = ASK_LIMIT
		}
		return 0, NotEnoughStocksError{currentAllowedQty}
	}

	l.Debugf("Check2: Passed.")
	orderPrice := getOrderFeePrice(ask.Price, ask.StockId, ask.OrderType)
	orderFee := getOrderFee(ask.StockQuantity, orderPrice)
	cashLeft := int64(user.Cash) - int64(orderFee)

	l.Debugf("Check3: User has %d cash currently. Will be left with %d cash after trade.", user.Cash, cashLeft)

	if cashLeft < MINIMUM_CASH_LIMIT {
		l.Debugf("Check3: Failed. Not enough cash.")
		return 0, NotEnoughCashError{}
	}

	l.Debugf("Check3: Passed. Creating Ask.")

	if err := createAsk(ask); err != nil {
		l.Errorf("Error creating the ask %+v", err)
		return 0, err
	}

	l.Infof("Created Ask order. AskId: ", ask.Id)

	db := getDB()
	tx := db.Begin()

	oldCash := user.Cash

	var errorHelper = func(format string, args ...interface{}) (uint32, error) {
		l.Errorf(format, args...)
		user.Cash = oldCash
		return 0, err
	}

	if err := SubtractOrderFee(user, orderFee, tx); err != nil {
		return errorHelper("Error while subtracting order fee from user. Rolling back. Error: %+v", err)
	}

	orderFeeTransaction := GetTransactionRef(
		userId,
		ask.StockId,
		OrderFeeTransaction,
		0,
		0,
		int64(-orderFee),
		utils.GetCurrentTimeISO8601(),
	)

	if err := tx.Save(orderFeeTransaction).Error; err != nil {
		return errorHelper("Error saving OrderFeeTransaction. Rolling back. Error: %+v", err)
	}

	l.Info("Saved OrderFeeTransaction for bid %d", ask.Id)

	if err := tx.Commit().Error; err != nil {
		return errorHelper("Error committing the transaction. Failing. %+v", err)
	}

	l.Info("Commited successfully for bid %d", ask.Id)

	// Update datastreams to add newly placed order in OpenOrders
	go func(ask *Ask, orderFeeTransaction *Transaction) {
		myOrdersStream := datastreamsManager.GetMyOrdersStream()
		transactionsStream := datastreamsManager.GetTransactionsStream()

		myOrdersStream.SendOrder(ask.UserId, &datastreams_pb.MyOrderUpdate{
			Id:            ask.Id,
			IsAsk:         true,
			IsNewOrder:    true,
			StockId:       ask.StockId,
			OrderPrice:    ask.Price,
			OrderType:     ask.ToProto().OrderType,
			StockQuantity: ask.StockQuantity,
		})
		transactionsStream.SendTransaction(orderFeeTransaction.ToProto())

		l.Infof("Sent through the datastreams")
	}(ask, orderFeeTransaction)

	return ask.Id, nil
}

// PlaceBidOrder places a Bid order for the user.
// The method is thread-safe like other exported methods of this package.
//
// Possible outcomes:
// 	1. Bid gets placed successfully
//  2. BidLimitExceededError is returned
//  3. NotEnoughCashError is returned
//  4. Other error is returned (e.g. if Database connection doesn't open)
func PlaceBidOrder(userId uint32, bid *Bid) (uint32, error) {
	l := logger.WithFields(logrus.Fields{
		"method":    "PlaceBidOrder",
		"param_id":  userId,
		"param_bid": fmt.Sprintf("%+v", bid),
	})

	l.Infof("PlaceBidOrder requested")

	// Place cap on order price only for limit orders
	if bid.OrderType == Limit {
		if bid.Price <= MINIMUM_ORDER_PRICE {
			l.Debugf("Minimum price check failed for ask order")
			return 0, MinimumPriceThresholdError{}
		}

		l.Debugf("Acquiring lock for bid order threshold check with stock id : %v ", bid.StockId)

		allStocks.m[bid.StockId].RLock()
		currentPrice := allStocks.m[bid.StockId].stock.CurrentPrice
		allStocks.m[bid.StockId].RUnlock()

		l.Debugf("Releasing lock for bid order threshold check with stock id : %v ", bid.StockId)

		var upperLimit = uint64((1 + ORDER_PRICE_WINDOW/100.0) * float64(currentPrice))
		var lowerLimit = uint64((1 - ORDER_PRICE_WINDOW/100.0) * float64(currentPrice))

		if bid.Price > upperLimit || bid.Price < lowerLimit {
			l.Debugf("Threshold price check failed for bid order")
			return 0, OrderPriceOutOfWindowError{currentPrice}
		}
	}

	l.Debugf("Acquiring exclusive write on user")
	ch, user, err := getUserExclusively(userId)
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
	if bid.StockQuantity > BID_LIMIT || bid.StockQuantity < 1 {
		l.Debugf("Check1: Failed.")
		return 0, OrderStockLimitExceeded{}
	}
	l.Debugf("Check1: Passed.")

	// Second Check: User should have enough cash
	orderPrice := getOrderFeePrice(bid.Price, bid.StockId, bid.OrderType)
	orderFee := getOrderFee(bid.StockQuantity, orderPrice)
	cashLeft := int64(user.Cash) - int64(bid.StockQuantity*bid.Price+orderFee)

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

	db := getDB()
	tx := db.Begin()

	oldCash := user.Cash

	var errorHelper = func(format string, args ...interface{}) (uint32, error) {
		l.Errorf(format, args...)
		user.Cash = oldCash
		return 0, err
	}

	if err := SubtractOrderFee(user, orderFee, tx); err != nil {
		return errorHelper("Error while subtracting order fee from user. Rolling back. Error: %+v", err)
	}

	// Update datastreams to add newly placed order in OpenOrders
	orderFeeTransaction := GetTransactionRef(
		userId,
		bid.StockId,
		OrderFeeTransaction,
		0,
		0,
		int64(-orderFee),
		utils.GetCurrentTimeISO8601(),
	)

	if err := tx.Save(orderFeeTransaction).Error; err != nil {
		return errorHelper("Error saving OrderFeeTransaction. Rolling back. Error: %+v", err)
	}

	l.Info("Saved OrderFeeTransaction for bid %d", bid.Id)

	if err := tx.Commit().Error; err != nil {
		return errorHelper("Error committing the transaction. Failing. %+v", err)
	}

	l.Info("Commited successfully for bid %d", bid.Id)

	// Update datastreams to add newly placed order in OpenOrders
	go func(bid *Bid, orderFeeTransaction *Transaction) {
		myOrdersStream := datastreamsManager.GetMyOrdersStream()
		transactionsStream := datastreamsManager.GetTransactionsStream()

		myOrdersStream.SendOrder(bid.UserId, &datastreams_pb.MyOrderUpdate{
			Id:            bid.Id,
			IsAsk:         false,
			IsNewOrder:    true,
			StockId:       bid.StockId,
			OrderPrice:    bid.Price,
			OrderType:     bid.ToProto().OrderType,
			StockQuantity: bid.StockQuantity,
		})
		transactionsStream.SendTransaction(orderFeeTransaction.ToProto())

		l.Infof("Sent through the datastreams")
	}(bid, orderFeeTransaction)

	return bid.Id, nil
}

// CancelOrder cancels a user's order. It'll check if the user was the one who placed it.
// It returns the pointer to Ask/Bid (whichever it was - the other is nil) that got cancelled.
// This pointer will have to be passed to the CancelOrder of the matching engine to remove it
// from there.
func CancelOrder(userId uint32, orderId uint32, isAsk bool) (*Ask, *Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "CancelOrder",
		"userId":  userId,
		"orderId": orderId,
		"isAsk":   isAsk,
	})

	l.Infof("CancelOrder requested")

	l.Debugf("Acquiring exclusive write on user")

	ch, _, err := getUserExclusively(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return nil, nil, err
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
			return nil, nil, InvalidOrderIDError{}
		} else if err != nil {
			l.Errorf("Unknown error in getAsk: %+v", err)
			return nil, nil, err
		}

		err = askOrder.Close()
		// don't log if order is already closed
		if _, ok := err.(AlreadyClosedError); !ok {
			l.Errorf("Unknown error while saving that ask is cancelled %+v", err)
		}
		// return the error anyway
		if err != nil {
			return nil, nil, err
		}

		l.Infof("Cancelled order")
		return askOrder, nil, nil
	} else {
		l.Debugf("Acquiring lock on bid order")
		bidOrder, err := getBid(orderId)
		if bidOrder == nil || bidOrder.UserId != userId {
			l.Errorf("Invalid bid id provided")
			return nil, nil, InvalidOrderIDError{}
		} else if err != nil {
			l.Errorf("Unknown error in getBid")
			return nil, nil, err
		}

		err = bidOrder.Close()
		// don't log if order is already closed
		if _, ok := err.(AlreadyClosedError); !ok {
			l.Errorf("Unknown error while saving that bid is cancelled %+v", err)
		}
		// return the error anyway
		if err != nil {
			return nil, nil, err
		}

		l.Infof("Cancelled order")
		return nil, bidOrder, nil
	}
}

func PerformBuyFromExchangeTransaction(userId uint32, stockId uint32, stockQuantity uint64) (*Transaction, error) {
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
	ch, user, err := getUserExclusively(userId)
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
		StockQuantity: int64(stockQuantityRemoved),
		Price:         price,
		Total:         -int64(price * stockQuantityRemoved),
		CreatedAt:     utils.GetCurrentTimeISO8601(),
	}

	oldCash := user.Cash
	user.Cash = uint64(int64(user.Cash) + transaction.Total)

	oldStocksInExchange := stock.StocksInExchange
	oldStocksInMarket := stock.StocksInMarket
	oldUpdatedAt := stock.UpdatedAt
	stock.StocksInExchange -= stockQuantityRemoved
	stock.StocksInMarket += stockQuantityRemoved
	stock.UpdatedAt = utils.GetCurrentTimeISO8601()

	/* Committing to database */
	db := getDB()

	tx := db.Begin()

	var errorHelper = func(format string, args ...interface{}) (*Transaction, error) {
		l.Errorf(format, args...)
		user.Cash = oldCash
		stock.StocksInExchange = oldStocksInExchange
		stock.StocksInMarket = oldStocksInMarket
		stock.UpdatedAt = oldUpdatedAt
		tx.Rollback()
		return nil, fmt.Errorf(format, args...)
	}

	if err := tx.Save(transaction).Error; err != nil {
		return errorHelper("Error creating the transaction. Rolling back. Error: %+v", err)
	}

	l.Debugf("Added transaction to Transactions table")

	if err := tx.Save(user).Error; err != nil {
		return errorHelper("Error deducting the cash from user's account. Rolling back. Error: %+v", err)
	}

	l.Debugf("Deducted cash from user's account. New balance: %d", user.Cash)

	if err := tx.Save(stock).Error; err != nil {
		return errorHelper("Error transferring stocks from exchange to market. Rolling back.")
	}

	l.Debugf("Transferred stocks from Exchange to Market")

	if err := tx.Commit().Error; err != nil {
		return errorHelper("Error committing the transaction. Failing. %+v", err)
	}

	l.Infof("Committed transaction. Removed %d stocks @ %d per stock. Total cost = %d. New balance: %d", stockQuantityRemoved, price, price*stockQuantityRemoved, user.Cash)

	go func(inExchange, inMarket uint64) {
		stockExchangeStream := datastreamsManager.GetStockExchangeStream()
		transactionsStream := datastreamsManager.GetTransactionsStream()

		stockExchangeStream.SendStockExchangeUpdate(stockId, &datastreams_pb.StockExchangeDataPoint{
			Price:            price,
			StocksInExchange: inExchange,
			StocksInMarket:   inMarket,
		})
		transactionsStream.SendTransaction(transaction.ToProto())

		l.Infof("Sent through the datastreams")
	}(stock.StocksInExchange, stock.StocksInMarket)

	return transaction, nil
}

/*
*	PerformOrderFillTransaction performs the following function
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

type AskOrderFillStatus uint8
type BidOrderFillStatus uint8

const (
	AskAlreadyClosed AskOrderFillStatus = iota // Order was already cancelled before execution started!
	AskDone                                    // Order either got fulfilled or dropped
	AskUndone                                  // Order yet to complete
)

const (
	BidAlreadyClosed BidOrderFillStatus = iota // Order was already cancelled before execution started!
	BidDone                                    // Order either got fulfilled or dropped
	BidUndone                                  // Order yet to complete
)

func PerformOrderFillTransaction(ask *Ask, bid *Bid, stockTradePrice uint64, stockTradeQty uint64) (AskOrderFillStatus, BidOrderFillStatus, *Transaction) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "PerformOrderFillTransaction",
		"askingUserId":  ask.UserId,
		"biddingUserId": bid.UserId,
		"stockId":       ask.StockId,
	})

	l.Infof("Attempting")

	/*
		if ask.isclosed() return askDone, bidNotDone
		if bid.isClosed() return askNotDone, bidDone

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

	/*
		Grab lock on user pair before checking for order cancellation,
		since CancelOrder will otherwise have a race condition with this one
	*/
	done, askingUser, biddingUser, err := getUserPairExclusive(ask.UserId, bid.UserId)
	if err != nil {
		l.Errorf("Unable to acquire locks on the user pair: %+v", err)
		return AskUndone, BidUndone, nil
	}
	defer close(done)

	/* Check if either (or both) of the order(s) has been closed already */
	askStatus := AskUndone
	bidStatus := BidUndone

	if ask.IsClosed {
		askStatus = AskAlreadyClosed
	}
	if bid.IsClosed {
		bidStatus = BidAlreadyClosed
	}

	if askStatus == AskAlreadyClosed || bidStatus == BidAlreadyClosed {
		l.Infof("Done. One of the orders already closed. %+s and %+s", askStatus, bidStatus)
		return askStatus, bidStatus, nil
	}

	/* We're here, so both orders are open now */

	var updateDataStreams = func(askTrans, bidTrans *Transaction) {
		myOrdersStream := datastreamsManager.GetMyOrdersStream()
		transactionsStream := datastreamsManager.GetTransactionsStream()

		if stockTradeQty != 0 || ask.IsClosed {
			myOrdersStream.SendOrder(ask.UserId, &datastreams_pb.MyOrderUpdate{
				Id:            ask.Id,
				IsAsk:         true,
				StockQuantity: ask.StockQuantity,
				TradeQuantity: stockTradeQty,
				IsClosed:      ask.IsClosed,
			})
		}
		if stockTradeQty != 0 || bid.IsClosed {
			myOrdersStream.SendOrder(bid.UserId, &datastreams_pb.MyOrderUpdate{
				Id:            bid.Id,
				IsAsk:         false,
				StockQuantity: bid.StockQuantity,
				TradeQuantity: stockTradeQty,
				IsClosed:      bid.IsClosed,
			})
		}

		if askTrans != nil {
			transactionsStream.SendTransaction(askTrans.ToProto())
			transactionsStream.SendTransaction(bidTrans.ToProto())
		}

		l.Infof("Sent through the datastreams")
	}

	//Check if bidder has enough cash
	var cashLeft = int64(biddingUser.Cash) - int64(stockTradePrice*stockTradeQty)

	if cashLeft < MINIMUM_CASH_LIMIT {
		l.Errorf("Check1: Failed. Not enough cash.")
		bid.Close()
		go updateDataStreams(nil, nil)
		go SendNotification(biddingUser.Id, fmt.Sprintf("Your Buy order#%d has been closed due to insufficient cash", bid.Id), false)
		return AskUndone, BidDone, nil
	}

	//Check if askingUser has enough stocks
	numStocks, err := getSingleStockCount(askingUser, ask.StockId)
	if err != nil {
		l.Errorf("Error getting stock count for askingUser: %+v", err)
		return AskUndone, BidUndone, nil
	}

	var numStocksLeft = numStocks - int64(stockTradeQty)

	if numStocksLeft < -SHORT_SELL_BORROW_LIMIT {
		l.Debugf("Check2: Failed. Not enough stocks.")
		currentAllowedQty := numStocks + SHORT_SELL_BORROW_LIMIT
		if currentAllowedQty > ASK_LIMIT {
			currentAllowedQty = ASK_LIMIT
		}
		ask.Close()
		go updateDataStreams(nil, nil)
		go SendNotification(askingUser.Id, fmt.Sprintf("Your Sell order#%d has been closed due to insufficient stocks", ask.Id), false)
		return AskDone, BidUndone, nil
	}

	//helper function to return a transaction object
	var makeTrans = func(userId uint32, stockId uint32, transType TransactionType, stockQty int64, price uint64, total int64) *Transaction {
		return &Transaction{
			UserId:        userId,
			StockId:       stockId,
			Type:          transType,
			StockQuantity: stockQty,
			Price:         price,
			Total:         total,
			CreatedAt:     utils.GetCurrentTimeISO8601(),
		}
	}

	total := int64(stockTradePrice * stockTradeQty)

	askTransaction := makeTrans(ask.UserId, ask.StockId, OrderFillTransaction, -int64(stockTradeQty), stockTradePrice, total)
	bidTransaction := makeTrans(bid.UserId, bid.StockId, OrderFillTransaction, int64(stockTradeQty), stockTradePrice, -total)

	// save old cash for rolling back
	askingUserOldCash := askingUser.Cash
	biddingUserOldCash := biddingUser.Cash

	//calculate user's updated cash
	askingUser.Cash += uint64(stockTradeQty) * stockTradePrice
	biddingUser.Cash -= uint64(stockTradeQty) * stockTradePrice

	// in case things go wrong and we've to roll back
	oldAskStockQuantityFulfilled := ask.StockQuantityFulfilled
	oldBidStockQuantityFulfilled := bid.StockQuantityFulfilled
	oldAskIsClosed := ask.IsClosed
	oldBidIsClosed := bid.IsClosed

	//calculate StockQuantityFulfilled and IsClosed for ask and bid order
	ask.StockQuantityFulfilled += uint64(stockTradeQty)
	bid.StockQuantityFulfilled += uint64(stockTradeQty)
	ask.IsClosed = ask.StockQuantity == ask.StockQuantityFulfilled
	bid.IsClosed = bid.StockQuantity == bid.StockQuantityFulfilled

	//Committing to database
	db := getDB()

	//Begin transaction
	tx := db.Begin()

	var errorHelper = func(fmt string, args ...interface{}) (AskOrderFillStatus, BidOrderFillStatus, *Transaction) {
		l.Errorf(fmt, args...)
		askingUser.Cash = askingUserOldCash
		biddingUser.Cash = biddingUserOldCash

		ask.StockQuantityFulfilled = oldAskStockQuantityFulfilled
		bid.StockQuantityFulfilled = oldBidStockQuantityFulfilled

		ask.IsClosed = oldAskIsClosed
		bid.IsClosed = oldBidIsClosed

		tx.Rollback()
		return AskUndone, BidUndone, nil
	}

	//save askTransaction
	if err := tx.Save(askTransaction).Error; err != nil {
		return errorHelper("Error creating the askTransaction. Rolling back. Error: %+v", err)
	}
	l.Debugf("Added askTransaction to Transactions table")

	//save bidTransaction
	if err := tx.Save(bidTransaction).Error; err != nil {
		return errorHelper("Error creating the bidTransaction. Rolling back. Error: %+v", err)
	}
	l.Debugf("Added bidTransaction to Transactions table")

	//update askingUser
	if err := tx.Save(askingUser).Error; err != nil {
		return errorHelper("Error updating askingUser.Cash Rolling back. Error: %+v", err)
	}

	//update biddingUserCash
	if err := tx.Save(biddingUser).Error; err != nil {
		return errorHelper("Error updating biddingUser.Cash Rolling back. Error: %+v", err)
	}

	//update StockQuantityFulfilled and IsClosed for ask order
	if err := tx.Save(ask).Error; err != nil {
		return errorHelper("Error updating ask.{StockQuantityFulfilled,IsClosed}. Rolling back. Error: %+v", err)
	}

	//update StockQuantityFulfilled and IsClosed for bid order
	if err := tx.Save(bid).Error; err != nil {
		return errorHelper("Error updating bid.{StockQuantityFulfilled,IsClosed}. Rolling back. Error: %+v", err)
	}

	// insert an OrderFill
	of := &OrderFill{
		AskId:         ask.Id,
		BidId:         bid.Id,
		TransactionId: askTransaction.Id, // We'll always store Ask
	}
	if err := tx.Save(of).Error; err != nil {
		return errorHelper("Error saving an orderfill. Rolling back. Error: %+v", err)
	}

	//Commit transaction
	if err := tx.Commit().Error; err != nil {
		l.Errorf("Error committing the transaction. Failing. %+v", err)
		return AskUndone, BidUndone, nil
	}

	go updateDataStreams(askTransaction, bidTransaction)

	UpdateStockVolume(ask.StockId, stockTradeQty)
	l.Infof("Transaction committed successfully. Traded %d at %d per stock. Total %d.", stockTradeQty, stockTradePrice, total)

	if err := UpdateStockPrice(ask.StockId, stockTradePrice); err != nil {
		l.Errorf("Error updating stock price. BUT SUPRRESSING IT.")
	}

	/* Set the order statuses */

	if ask.IsClosed {
		askStatus = AskDone
	} else {
		askStatus = AskUndone
	}

	if bid.IsClosed {
		bidStatus = BidDone
	} else {
		bidStatus = BidUndone
	}

	return askStatus, bidStatus, askTransaction
}

var MortgagePutLimitRWMutex sync.RWMutex
var MortgagePutLimit int64 = 4000000

type WayTooMuchCashError struct {
}

func (e WayTooMuchCashError) Error() string {
	return "You already have more than enough cash. You cannot mortgage stocks right now."
}

// PerformMortgageTransaction returns mortgage transaction and/or error
func PerformMortgageTransaction(userId, stockId uint32, stockQuantity int64, retrievePrice uint64) (*Transaction, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "PerformMortgageTransaction",
		"param_userId":        userId,
		"param_stockId":       stockId,
		"param_stockQuantity": stockQuantity,
		"param_retrievePrice": retrievePrice,
	})

	l.Infof("PerformMortgageTransaction requested")

	l.Debugf("Acquiring exclusive write on user")
	ch, user, err := getUserExclusively(userId)
	if err != nil {
		l.Errorf("Errored : %+v ", err)
		return nil, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	/* Committing to database */
	db := getDB()
	tx := db.Begin()

	allStocks.m[stockId].RLock()
	mortgagePrice := allStocks.m[stockId].stock.AvgLastPrice
	allStocks.m[stockId].RUnlock()

	l.Infof("Taking current price of stock as %d", mortgagePrice)

	var trTotal int64

	if stockQuantity >= 0 {
		mortgagePrice = retrievePrice
		trTotal, err = retrieveStocksAction(user.Id, stockId, stockQuantity, user.Cash, retrievePrice, tx)
	} else {
		// Sending stockQuantity negative as stockQuantity itself is negative makeing -stockQuantity
		trTotal, err = mortgageStocksAction(user, stockId, -stockQuantity, mortgagePrice, tx)
	}

	if err != nil {
		tx.Rollback()
		l.Error(err)
		return nil, err
	}

	transaction := &Transaction{
		UserId:        userId,
		StockId:       stockId,
		Type:          MortgageTransaction,
		StockQuantity: stockQuantity,
		Price:         mortgagePrice,
		Total:         trTotal,
		CreatedAt:     utils.GetCurrentTimeISO8601(),
	}

	// A lock on user and stock has been acquired.
	// Safe to make changes to this user and this stock

	oldCash := user.Cash
	user.Cash = uint64(int64(user.Cash) + trTotal)

	errorHelper := func(format string, args ...interface{}) (*Transaction, error) {
		l.Errorf(format, args...)
		user.Cash = oldCash
		tx.Rollback()
		return nil, fmt.Errorf(format, args...)
	}

	if err := tx.Save(transaction).Error; err != nil {
		return errorHelper("Error creating the transaction. Rolling back. Error: %+v", err)
	}

	l.Debugf("Added transaction to Transactions table")

	if err := tx.Save(user).Error; err != nil {
		return errorHelper("Error updating user's cash. Rolling back. Error: %+v", err)
	}

	l.Infof("Updated user's cash. New balance: %d", user.Cash)

	if err := tx.Commit().Error; err != nil {
		return errorHelper("Error committing the transaction. Failing. %+v", err)
	}

	l.Debugf("Committed transaction. Success.")

	go func(transaction Transaction) {
		transactionsStream := datastreamsManager.GetTransactionsStream()
		transactionsStream.SendTransaction(transaction.ToProto())
		l.Infof("Sent through the datastreams")
	}(*transaction)

	return transaction, nil
}

func GetStocksOwned(userId uint32) (map[uint32]int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetStocksOwned",
		"userId": userId,
	})

	l.Info("GetStocksOwned requested")

	l.Debugf("Acquiring lock on user")

	ch, _, err := getUserExclusively(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return nil, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released lock on user")
	}()

	db := getDB()

	sql := "Select stockId, sum(stockQuantity) as stockQuantity from Transactions where userId=? group by stockId"
	rows, err := db.Raw(sql, userId).Rows()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer rows.Close()

	stocksOwned := make(map[uint32]int64)
	for rows.Next() {
		var stockId uint32
		var stockQty int64
		rows.Scan(&stockId, &stockQty)

		stocksOwned[stockId] = stockQty
	}

	return stocksOwned, nil
}

// Logout removes the user from RAM.
func Logout(userID uint32) {
	userLocks.Lock()
	delete(userLocks.m, userID)
	userLocks.Unlock()
}
