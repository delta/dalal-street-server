package models

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
	"github.com/delta/dalal-street-server/templates"
	"github.com/delta/dalal-street-server/utils"
	"github.com/jinzhu/gorm"

	"github.com/sirupsen/logrus"
	//"github.com/satori/go.uuid"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
)

var (
	UnauthorizedError             = errors.New("Invalid credentials")
	NotRegisteredError            = errors.New("Not registered")
	AlreadyRegisteredError        = errors.New("Already registered")
	UnverifiedUserError           = errors.New("User has not verified account")
	TemporaryPasswordExpiredError = errors.New("Temporary password has expired")
	PasswordMismatchError         = errors.New("Passwords mismatch error")
	PragyanUserError              = errors.New("Pragyan user error")
	InvalidTemporaryPasswordError = errors.New("Invalid temporary password")
	UserNotFoundError             = errors.New("Invalid userId.")
	InvalidReferralCodeError      = errors.New("Invalid Referral Code")
	/*
		Net worth <= 0 => tax percentage = 0%
		0 < Net worth <= 100000 => tax percentage = 2%
		100000 < Net worth <= 500000 => tax percentage = 5%
		500000 < Net worth <= 1000000 => tax percentage = 9%
		1000000 < Net worth <= 2000000 => tax percentage = 15%
		Net worth > 2000000 => tax percentage = 25%
	*/
	TaxBrackets = map[int64]uint64{
		0:       0,
		100000:  2,
		500000:  5,
		1000000: 9,
		2000000: 15,
	}
	MaxTaxPercent uint64 = 25
)

var TotalUserCount uint32

// User models the User object.
type User struct {
	Id              uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	Email           string `gorm:"unique;not null" json:"email"`
	Name            string `gorm:"not null" json:"name"`
	Cash            uint64 `gorm:"not null" json:"cash"`
	Total           int64  `gorm:"not null" json:"total"`
	CreatedAt       string `gorm:"column:createdAt;not null" json:"created_at"`
	IsHuman         bool   `gorm:"column:isHuman;not null" json:"is_human"`
	ReservedCash    uint64 `gorm:"column:reservedCash;not null" json:"reserved_cash"`
	IsPhoneVerified bool   `gorm:"column:isPhoneVerified;not null" json:"is_phone_verified"`
	IsAdmin         bool   `gorm:"column:isAdmin;not null" json:"is_admin"`
	IsOTPBlocked    bool   `gorm:"column:isOtpBlocked;not null" json:"is_otp_blocked"`
	OTPRequestCount int64  `gorm:"column:otpRequestCount;not null" json:"otp_request_count"`
	IsBlocked       bool   `gorm:"column:isBlocked;not null" json:"is_blocked"`
	BlockCount      int64  `gorm:"column:blockCount;not null" json:"block_count"`
}

func (u *User) ToProto() *models_pb.User {
	return &models_pb.User{
		Id:              u.Id,
		Email:           u.Email,
		Name:            u.Name,
		Cash:            u.Cash,
		Total:           u.Total,
		CreatedAt:       u.CreatedAt,
		IsHuman:         u.IsHuman,
		ReservedCash:    u.ReservedCash,
		IsPhoneVerified: u.IsPhoneVerified,
		IsAdmin:         u.IsAdmin,
		IsOtpBlocked:    u.IsOTPBlocked,
		OtpRequestCount: u.OTPRequestCount,
		IsBlocked:       u.IsBlocked,
		BlockCount:      u.BlockCount,
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
		"method":      "Login",
		"param_email": email,
	})

	l.Infof("Attempting to login user")

	db := getDB()

	var registeredUser = Registration{
		Email: email,
	}

	err := db.Table("Registrations").Where("email = ?", email).First(&registeredUser).Error
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

	// Check if user has been verified or not only on docker
	if config.Stage == "docker" && registeredUser.IsVerified == false {
		l.Errorf("User (%s) attempted login before verification", email)
		return User{}, UnverifiedUserError
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
func RegisterUser(email, password, fullName, referralCode string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":      "Register",
		"param_email": email,
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

	password, _ = hashPassword(password)
	verificationKey, _ := getVerificationKey(email)
	register := &Registration{
		Email:                  email,
		Password:               password,
		IsPragyan:              false,
		IsVerified:             false,
		Name:                   fullName,
		VerificationKey:        verificationKey,
		VerificationEmailCount: 1,
	}

	if referralCode != "" {
		// User has entered a valid referralCode
		l.Errorf("the referral code is : %v\n\n", referralCode)
		codeID, err := VerifyReferralCode(referralCode)
		if codeID != 0 {
			(*register).ReferralCodeID = codeID
		} else if codeID == 0 && err == nil {
			// invalid referralCode
			return InvalidReferralCodeError
		} else {
			// error while verifying referralcode
			return err
		}
	}

	u, err := createUser(fullName, email)
	if err != nil {
		l.Errorf("Server error in Create user while logging in Pragyan user for the first time: %+v", err)
		return err
	}
	(*register).UserId = u.Id

	err = db.Save(register).Error
	if err != nil {
		return err
	}

	// Send verification email only if running on docker
	if config.Stage == "docker" {
		l.Debugf("Sending verification email to %s", email)
		verificationURL := fmt.Sprintf("https://dalal.pragyan.org/api/verify?key=%s", verificationKey)
		htmlContent := fmt.Sprintf(`%s
							%s
							%s`, templates.HtmlEmailVerificationTemplateHead, verificationURL, templates.HtmlEmailVerificationTemplateTail)
		plainContent := fmt.Sprintf(templates.PlainEmailVerificationTemplate, verificationURL)
		err = utils.SendEmail("noreply@dalal.pragyan.org", "Account Verification", email, plainContent, htmlContent)
		if err != nil {
			l.Errorf("Error while sending verification email to player %s", err)
			return err
		}
	}

	return nil
}

func hashPassword(password string) (string, error) {
	h := sha1.New()
	h.Write([]byte(password))
	sha1Hash := hex.EncodeToString(h.Sum(nil))
	return sha1Hash, nil
}

func getVerificationKey(email string) (string, error) {
	key := fmt.Sprintf("%s %s", email, os.Getenv("SECRET_KEY"))
	h := sha1.New()
	h.Write([]byte(key))
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
		Email:           email,
		Name:            name,
		Cash:            STARTING_CASH,
		Total:           STARTING_CASH,
		CreatedAt:       utils.GetCurrentTimeISO8601(),
		IsHuman:         true,
		ReservedCash:    0,
		IsPhoneVerified: false,
		IsOTPBlocked:    false,
		OTPRequestCount: 0,
		IsBlocked:       false,
		BlockCount:      0,
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
	})

	form := url.Values{
		"user_email":   {email},
		"user_pass":    {password},
		"event_id":     {config.EventId},
		"event_secret": {config.EventSecret},
	}

	l.Debugf("Attempting login to Pragyan")
	resp, err := http.PostForm("https://api.pragyan.org/21/event/login", form)
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

//getTotalSingleStockCount returns total stocks a user owns including reserved stocks
func getTotalSingleStockCount(u *User, stockId uint32) (int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "getTotalSingleStockCount",
		"param_u":       fmt.Sprintf("%+v", u),
		"param_stockId": stockId,
	})

	l.Debugf("Attempting")

	db := getDB()

	var stockCount = struct{ Tsc int64 }{0}

	sql := "Select sum(StockQuantity) + sum(ReservedStockQuantity) as tsc from Transactions where UserId=? and StockId=?"
	if err := db.Raw(sql, u.Id, stockId).Scan(&stockCount).Error; err != nil {
		l.Error(err)
		return 0, err
	}

	l.Debugf("Got %d", stockCount.Tsc)

	return stockCount.Tsc, nil
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

// NotEnoughActualWorth is generated when an Ask's StockQuantity is such that
// deducting those many stocks will mean that with the current stock worth
// multiplied by number of stocks being short sold is more than the cash in
// hand and stock worth of the user
type NotEnoughActualWorthError struct {
	actualWorth int64
}

func (e NotEnoughActualWorthError) Error() string {
	return fmt.Sprintf("Not have actual networh to place this order. Cash in hand plus stock worth should be atleast: %d", e.actualWorth)
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

// InvalidTransaction is given out when a user tries to cancel an order he didn't make or that didn't exist
type InvalidTransaction struct{}

func (e InvalidTransaction) Error() string {
	return fmt.Sprintf("No such transaction found")
}

// InvalidDividendAmountError is given out when the dividend amount in negative.
type InvalidDividendAmountError struct{}

func (e InvalidDividendAmountError) Error() string {
	return fmt.Sprintf("Invalid dividend amount")
}

// InvalidStockIdError is given out when the stock id is either more than the number of stocks or negative.
type InvalidStockIdError struct{}

func (e InvalidStockIdError) Error() string {
	return fmt.Sprintf("Invalid stock id")
}

// StockBankruptError is given out when the stock is already bankrupt.
type StockBankruptError struct {
}

func (e StockBankruptError) Error() string {
	return fmt.Sprintf("This stock is already bankrupt you are not allowed to perform any action on this stock.")
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
			return nil, nil, UserNotFoundError
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

// SubtractUserCash subtracts cash for users account
// This function should be called ONLY AFTER lock is obtained on user
func SubtractUserCash(user *User, fee uint64, tx *gorm.DB) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "SubtractUserCash",
		"param_user": fmt.Sprintf("%+v", user),
		"param_fee":  fmt.Sprintf("%d", fee),
	})

	user.Cash = user.Cash - fee

	if err := tx.Save(user).Error; err != nil {
		return err
	}

	l.Infof("Updated user cash. User now has %d", user.Cash)

	return nil
}

// AddUserReservedCash adds reservedCash to the users account
// This function should be called ONLY AFTER lock is obtained on user
func AddUserReservedCash(user *User, reservedCash uint64, tx *gorm.DB) error {
	l := logger.WithFields(logrus.Fields{
		"method":             "AddUserReservedCash",
		"param_user":         fmt.Sprintf("%+v", user),
		"param_reservedCash": fmt.Sprintf("%d", reservedCash),
	})

	user.ReservedCash = user.ReservedCash + reservedCash

	if err := tx.Save(user).Error; err != nil {
		return err
	}

	l.Infof("Updated user reserved cash. User now has %d", user.ReservedCash)

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

	l.Infof("PlaceAskOrder requested")
	if isBankrupt := IsStockBankrupt(ask.StockId); isBankrupt {
		l.Infof("Stock already bankrupt. Returning function.")
		return 0, StockBankruptError{}
	}

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

	var shortSellMin = numStocksLeft * int64(ask.Price)
	var stockWorth int64 = 0

	db := getDB()

	sql := "Select stockId, sum(stockQuantity) as stockQuantity from Transactions where userId=? group by stockId"
	rows, err := db.Raw(sql, userId).Rows()
	if err != nil {
		l.Error(err)
		return 0, err
	}
	defer rows.Close()

	stocksOwned := make(map[uint32]int64)
	for rows.Next() {
		var stockId uint32
		var stockQty int64
		rows.Scan(&stockId, &stockQty)

		stocksOwned[stockId] = stockQty
	}

	// Find Stock worth of user
	for id, number := range stocksOwned {
		allStocks.m[id].RLock()
		stockWorth = stockWorth + int64(allStocks.m[id].stock.CurrentPrice)*number
		allStocks.m[id].RUnlock()
	}

	//Actual worth of user includes only
	//Stock worth and cash
	//Does not include reserved cash and stocks
	var actualWorth = int64(user.Cash) + stockWorth

	l.Debugf("Check3: Current stocks: %d. Stocks after trade: %d. User Actual Worth(Cash in hand + Stock Worth) %d", numStocks, numStocksLeft, user.Total)

	//Check if networth of user is more than the number of stocks
	//which are short sold
	if numStocksLeft < 0 && -(actualWorth) > shortSellMin {
		l.Debugf("Check3: Failed. Not enough actual worth to short sell.")
		return 0, NotEnoughActualWorthError{-shortSellMin}
	}

	l.Debugf("Check3: Passed.")

	orderPrice := getOrderFeePrice(ask.Price, ask.StockId, ask.OrderType)
	orderFee := getOrderFee(ask.StockQuantity, orderPrice)
	cashLeft := int64(user.Cash) - int64(orderFee)

	l.Debugf("Check4: User has %d cash currently. Will be left with %d cash after trade.", user.Cash, cashLeft)

	if cashLeft < MINIMUM_CASH_LIMIT {
		l.Debugf("Check4: Failed. Not enough cash.")
		return 0, NotEnoughCashError{}
	}

	l.Debugf("Check4: Passed. Creating Ask.")

	oldCash := user.Cash

	db = getDB()
	tx := db.Begin()

	var errorHelper = func(format string, args ...interface{}) (uint32, error) {
		l.Errorf(format, args...)
		user.Cash = oldCash
		tx.Rollback()
		return 0, err
	}

	if err := createAsk(ask, tx); err != nil {
		return errorHelper("Error while creating Ask. Rolling back. Error: %+v", err)
	}

	l.Infof("Created Ask order. AskId: %d", ask.Id)

	if err := SubtractUserCash(user, orderFee, tx); err != nil {
		return errorHelper("Error while subtracting order fee from user. Rolling back. Error: %+v", err)
	}

	orderFeeTransaction := GetTransactionRef(
		userId,
		ask.StockId,
		OrderFeeTransaction,
		0,
		0,
		0,
		0,
		int64(-orderFee),
	)

	if err := tx.Save(orderFeeTransaction).Error; err != nil {
		return errorHelper("Error saving OrderFeeTransaction. Rolling back. Error: %+v", err)
	}

	l.Infof("Saved OrderFeeTransaction for bid %d", ask.Id)

	// Going to reserve stocks for this order
	placeOrderTransaction := GetTransactionRef(
		userId,
		ask.StockId,
		PlaceOrderTransaction,
		int64(ask.StockQuantity),
		-1*int64(ask.StockQuantity),
		0,
		0,
		0,
	)

	l.Infof("Reserving stocks for ask %d", ask.Id)

	if err := savePlaceOrderTransaction(ask.Id, placeOrderTransaction, true, tx); err != nil {
		return errorHelper("Error reserving stocks. Rolling back. Error: %+v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return errorHelper("Error committing the transaction. Failing. %+v", err)
	}

	l.Infof("Commited successfully for bid %d", ask.Id)

	// Update datastreams to add newly placed order in OpenOrders
	go func(ask *Ask, orderFeeTransaction, placeOrderTransaction *Transaction) {
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
		transactionsStream.SendTransaction(placeOrderTransaction.ToProto())

		l.Infof("Sent through the datastreams")
	}(ask, orderFeeTransaction, placeOrderTransaction)

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
	if isBankrupt := IsStockBankrupt(bid.StockId); isBankrupt {
		l.Infof("Stock already bankrupt. Returning function.")
		return 0, StockBankruptError{}
	}

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
	cashLeft := int64(user.Cash) - int64(bid.StockQuantity*orderPrice+orderFee)

	reservedCash := uint64(bid.StockQuantity * orderPrice)
	l.Debugf("Cash to be reserved is %d", reservedCash)
	l.Debugf("Check2: User has %d cash currently. Will be left with %d cash after trade.", user.Cash, cashLeft)

	if cashLeft < MINIMUM_CASH_LIMIT {
		l.Debugf("Check2: Failed. Not enough cash.")
		return 0, NotEnoughCashError{}
	}

	l.Debugf("Check2: Passed. Creating Bid")

	db := getDB()
	tx := db.Begin()

	oldCash := user.Cash
	oldReservedCash := user.ReservedCash

	var errorHelper = func(format string, args ...interface{}) (uint32, error) {
		l.Errorf(format, args...)
		user.Cash = oldCash
		user.ReservedCash = oldReservedCash
		tx.Rollback()
		return 0, err
	}

	if err := createBid(bid, tx); err != nil {
		return errorHelper("Error while creating bid. Rolling back. Error: %+v", err)
	}

	l.Infof("Created Bid order. BidId: %d", bid.Id)

	if err := SubtractUserCash(user, orderFee, tx); err != nil {
		return errorHelper("Error while subtracting order fee from user. Rolling back. Error: %+v", err)
	}

	if err := AddUserReservedCash(user, reservedCash, tx); err != nil {
		return errorHelper("Error while adding reserved cash to the user. Rolling back. Error: %+v", err)
	}

	// Update datastreams to add newly placed order in OpenOrders
	orderFeeTransaction := GetTransactionRef(
		userId,
		bid.StockId,
		OrderFeeTransaction,
		0,
		0,
		0,
		0,
		int64(-orderFee),
	)

	if err := tx.Save(orderFeeTransaction).Error; err != nil {
		return errorHelper("Error saving OrderFeeTransaction. Rolling back. Error: %+v", err)
	}

	l.Infof("Saved OrderFeeTransaction for bid. Now subtracting user cash for bid %d", bid.Id)

	if err := SubtractUserCash(user, bid.StockQuantity*orderPrice, tx); err != nil {
		return errorHelper("Error subtracting cash. Rolling back. Error: %+v", err)
	}

	// Going to reserve cash for this order
	placeOrderTransaction := GetTransactionRef(
		userId,
		bid.StockId,
		PlaceOrderTransaction,
		0,
		0,
		0,
		int64(bid.StockQuantity*orderPrice),
		-1*int64(bid.StockQuantity*orderPrice),
	)

	l.Infof("Reserving cash for bid %d", bid.Id)

	if err := savePlaceOrderTransaction(bid.Id, placeOrderTransaction, false, tx); err != nil {
		return errorHelper("Error reserving cash. Rolling back. Error: %+v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return errorHelper("Error committing the transaction. Failing. %+v", err)
	}

	l.Infof("Commited successfully for bid %d", bid.Id)

	// Update datastreams to add newly placed order in OpenOrders
	go func(bid *Bid, orderFeeTransaction, placeOrderTransaction *Transaction) {
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
		transactionsStream.SendTransaction(placeOrderTransaction.ToProto())

		l.Infof("Sent through the datastreams")
	}(bid, orderFeeTransaction, placeOrderTransaction)

	return bid.Id, nil
}

// saveAskCancelOrderTransaction creates a CancelOrderTransaction for Ask orders and pushses to stream
func saveAskCancelOrderTransaction(askOrder *Ask, user *User, tx *gorm.DB) error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "saveAskCancelOrderTransaction",
		"userId":  user.Id,
		"orderId": askOrder.Id,
	})

	cancelOrderTransaction := GetTransactionRef(
		user.Id,
		askOrder.StockId,
		CancelOrderTransaction,
		-1*int64(askOrder.StockQuantity-askOrder.StockQuantityFulfilled),
		int64(askOrder.StockQuantity-askOrder.StockQuantityFulfilled),
		0,
		0,
		0,
	)

	if err := tx.Save(cancelOrderTransaction).Error; err != nil {
		l.Errorf("Error while commiting %+v", err)
		return err
	}

	go func(cancelOrderTransaction *Transaction) {
		transactionsStream := datastreamsManager.GetTransactionsStream()
		transactionsStream.SendTransaction(cancelOrderTransaction.ToProto())

		l.Infof("Sent through the datastreams")
	}(cancelOrderTransaction)

	return nil
}

// saveBidCancelOrderTransaction creates a CancelOrderTransaction for Ask orders and pushses to stream
func saveBidCancelOrderTransaction(bidOrder *Bid, user *User, tx *gorm.DB) error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "saveBidCancelOrderTransaction",
		"userId":  user.Id,
		"orderId": bidOrder.Id,
	})

	reservedCash, _, err := getPlaceOrderTransactionDetails(bidOrder.Id, false)
	if err != nil {
		l.Errorf("Could not retrieve reserved cash. Error: %+v", err)
		return err
	}

	stocksFulfilled := bidOrder.StockQuantity - bidOrder.StockQuantityFulfilled
	reservedCash = int64(float64(reservedCash) * float64(stocksFulfilled) / float64(bidOrder.StockQuantity))
	cancelOrderTransaction := GetTransactionRef(
		user.Id,
		bidOrder.StockId,
		CancelOrderTransaction,
		0,
		0,
		0,
		-1*reservedCash,
		reservedCash,
	)

	user.Cash += uint64(reservedCash)
	user.ReservedCash -= uint64(reservedCash)

	if err := tx.Save(user).Error; err != nil {
		l.Errorf("Error while adding reserved cash back to user. Error: %+v", err)
		return err
	}

	if err := tx.Save(cancelOrderTransaction).Error; err != nil {
		l.Errorf("Error while saving cancelOrderTransaction %+v", err)
		return err
	}

	go func(cancelOrderTransaction *Transaction) {
		transactionsStream := datastreamsManager.GetTransactionsStream()
		transactionsStream.SendTransaction(cancelOrderTransaction.ToProto())

		l.Infof("Sent through the datastreams")
	}(cancelOrderTransaction)

	return nil
}

// CancelOrder cancels a user's order. It'll check if the user was the one who placed it.
// It returns the pointer to Ask/Bid (whichever it was - the other is nil) that got cancelled.
// This pointer will have to be passed to the CancelOrder of the matching engine to remove it
// from there. At the same time, whatever was reserved will be returned via a CancelOrderTransaction
func CancelOrder(userId uint32, orderId uint32, isAsk bool) (*Ask, *Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "CancelOrder",
		"userId":  userId,
		"orderId": orderId,
		"isAsk":   isAsk,
	})

	l.Infof("CancelOrder requested")

	l.Debugf("Acquiring exclusive write on user")

	ch, user, err := getUserExclusively(userId)
	if err != nil {
		l.Errorf("Errored: %+v", err)
		return nil, nil, err
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	db := getDB()
	tx := db.Begin()

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

		err = askOrder.Close(tx)
		// don't log if order is already closed
		if _, ok := err.(AlreadyClosedError); !ok {
			l.Errorf("Unknown error while saving that ask is cancelled %+v", err)
		}
		// return the error anyway
		if err != nil {
			tx.Rollback()
			return nil, nil, err
		}

		// Place CancelOrderTransaction to return stocks
		if err := saveAskCancelOrderTransaction(askOrder, user, tx); err != nil {
			l.Errorf("Error while trying to cancel ask order %d. Error: %+v", askOrder.Id, err)
			tx.Rollback()
			return nil, nil, err
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			l.Errorf("Error while commiting %+v", err)
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

		err = bidOrder.Close(tx)
		// don't log if order is already closed
		if _, ok := err.(AlreadyClosedError); !ok {
			l.Errorf("Unknown error while saving that bid is cancelled %+v", err)
		}
		// return the error anyway
		if err != nil {
			tx.Rollback()
			return nil, nil, err
		}

		oldCash := user.Cash
		oldReservedCash := user.ReservedCash

		// Place CancelOrderTransaction to return stocks
		if err := saveBidCancelOrderTransaction(bidOrder, user, tx); err != nil {
			user.Cash = oldCash
			user.ReservedCash = oldReservedCash
			tx.Rollback()
			l.Errorf("Error while cancelling bid order. Error: %+v", err)
			return nil, nil, err
		}

		if err := tx.Commit().Error; err != nil {
			user.Cash = oldCash
			user.ReservedCash = oldReservedCash
			tx.Rollback()
			l.Errorf("Error while commiting %+v", err)
			return nil, nil, err
		}

		l.Infof("Cancelled order")
		return nil, bidOrder, nil
	}
}

// Helper function to determine what percentage the user should be taxed
func getTaxPercent(netCash int64) uint64 {

	keys := make([]int64, 0)
	for taxBracket := range TaxBrackets {
		keys = append(keys, taxBracket)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for _, taxBracket := range keys {
		if netCash <= taxBracket {
			return TaxBrackets[taxBracket]
		}
	}
	return MaxTaxPercent
}

// Helper to calculate tax for bidding user.
// It is assumed that an Exclusive Read and Write lock is already obtained for the user.
// DO NOT call this function without obtaining the lock.
func getTaxForBiddingUser(tx *gorm.DB, stockId uint32, stockQuantity uint64, stockPrice uint64, biddingUser *User) (*TransactionSummary, *Transaction) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "getTaxForBiddingUser",
		"param_stockId":       stockId,
		"param_stockQuantity": stockQuantity,
		"param_stockPrice":    stockPrice,
		"param_biddingUser":   fmt.Sprintf("%+v", biddingUser),
	})

	userId := biddingUser.Id
	stockPriceFloat64 := float64(stockPrice)
	stockQuantityFloat64 := float64(stockQuantity)

	// Default value = 0
	var tax uint64

	transactionSummary := &TransactionSummary{UserId: userId, StockId: stockId}
	tx.Where("userId = ? AND stockId = ?", userId, stockId).First(&transactionSummary)

	l.Debugf("TransactionSummary object retrieved : %+v", transactionSummary)

	if transactionSummary.Id != 0 {
		// This user has already placed orders for this stock before
		currentStocksHeld := float64(transactionSummary.StockQuantity)
		if currentStocksHeld < 0 {
			// User is buying back his short sold stocks and so should be taxed

			taxableStockQty := math.Min(stockQuantityFloat64, -currentStocksHeld)
			/*
				if taxableStockQty = stockQuantity, then the user is buying back some of his short sold
				stocks, but not all of them

				if taxableStockQty = -currentStocksHeld, then the user is buying back all of his short sold stocks
				selling and then buying some more stocks on top of that
			*/

			profit := taxableStockQty * (transactionSummary.Price - stockPriceFloat64)
			if profit > 0 {
				netCash := biddingUser.Total
				tax = uint64(profit) * getTaxPercent(netCash) / 100
				biddingUser.Cash -= tax
				l.Debugf("Profit = %v. Tax = %v * %v / 100 = %v", profit, profit, getTaxPercent(netCash), tax)
			} else {
				l.Debugf("Profit = %v <= 0. Therefore, no tax.", profit)
			}
			transactionSummary.StockQuantity += int64(stockQuantity)
			if taxableStockQty == -currentStocksHeld {
				// User has bough back all of his short sold stocks and is now buying more, therefore the
				// price in the database should update to reflect this
				transactionSummary.Price = stockPriceFloat64
			}
			l.Debugf("User is buying back his short sold stocks. Database is going to be updated - %+v", transactionSummary)
		} else {
			// User already has some of these stocks and is buying more, therefore he shouldn't be taxed.
			// But update the record in the database
			transactionSummary.Price = ((currentStocksHeld * transactionSummary.Price) + (stockQuantityFloat64 * stockPriceFloat64)) / (currentStocksHeld + stockQuantityFloat64)
			transactionSummary.StockQuantity += int64(stockQuantity)
			l.Debugf("User is buying more stocks. No tax is added, but database is going to be updated - %+v", transactionSummary)
		}

	} else {
		// Effectively the first time this user is buying this stock.
		// Therefore, don't tax him. Just add the transaction into the database
		transactionSummary.StockQuantity = int64(stockQuantity)
		transactionSummary.Price = stockPriceFloat64
		l.Debugf("Effectively the first time user is buying these stocks. No tax is added, but database is going to be updated - %+v", transactionSummary)
	}

	taxTransaction := GetTransactionRef(userId, stockId, TaxTransaction, 0, 0, 0, 0, -int64(tax))

	if tax == 0 {
		return transactionSummary, nil
	}

	return transactionSummary, taxTransaction
}

// Helper to calculate tax for asking user
// It is assumed that an Exclusive Read and Write lock is already obtained for the user.
// DO NOT call this function without obtaining the lock.
func getTaxForAskingUser(tx *gorm.DB, stockId uint32, stockQuantity uint64, stockPrice uint64, askingUser *User) (*TransactionSummary, *Transaction) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "getTaxForAskingUser",
		"param_stockId":       stockId,
		"param_stockQuantity": stockQuantity,
		"param_stockPrice":    stockPrice,
		"param_biddingUser":   fmt.Sprintf("%+v", askingUser),
	})

	userId := askingUser.Id
	stockPriceFloat64 := float64(stockPrice)
	stockQuantityFloat64 := float64(stockQuantity)

	// Default value = 0
	var tax uint64

	transactionSummary := &TransactionSummary{UserId: userId, StockId: stockId}
	tx.Where("userId = ? AND stockId = ?", userId, stockId).First(&transactionSummary)

	l.Debugf("TransactionSummary object retrieved : %+v", transactionSummary)

	if transactionSummary.Id != 0 {
		// This user has already placed orders for this stock before

		currentStocksHeld := float64(transactionSummary.StockQuantity)
		if currentStocksHeld > 0 {
			// User is selling some stocks that he possesses, he must be taxed.

			taxableStockQty := math.Min(stockQuantityFloat64, currentStocksHeld)
			/*
				if taxableStockQty = stockQuantity, then the user is selling some of his existing stocks

				if taxableStockQty = currentStocksHeld, then the user is selling all of his existing
				stocks and then short selling (stockQuantity - currentStocksHeld) number of stocks
			*/

			profit := taxableStockQty * (stockPriceFloat64 - transactionSummary.Price)
			if profit > 0 {
				netCash := askingUser.Total
				tax = uint64(profit) * getTaxPercent(netCash) / 100
				askingUser.Cash -= tax
				l.Debugf("Profit = %v. Tax = %v * %v / 100 = %v", profit, profit, getTaxPercent(netCash), tax)
			} else {
				l.Debugf("Profit = %v <= 0. Therefore, no tax.", profit)
			}
			transactionSummary.StockQuantity -= int64(stockQuantity)
			if taxableStockQty == currentStocksHeld {
				// User has sold all of his stocks and is now short selling, therefore the price in the
				// database should update to reflect this
				transactionSummary.Price = stockPriceFloat64
			}
			l.Debugf("User is selling his stocks. Database is going to be updated - %+v", transactionSummary)
		} else {
			// User has already short sold this stock, and is now short selling more of it. Therefore,
			// no need to tax him. But update the record in the database.
			transactionSummary.Price = ((currentStocksHeld * transactionSummary.Price) - (stockQuantityFloat64 * stockPriceFloat64)) / (currentStocksHeld - stockQuantityFloat64)
			transactionSummary.StockQuantity -= int64(stockQuantity)
			l.Debugf("User is short selling more stocks. No tax is added, but database is going to be updated - %+v", transactionSummary)
		}

	} else {
		// Effectively the first time this user has placed an order for this stock and it is a short sell
		// Therefore, don't tax him. Just add the transaction into the database
		transactionSummary.StockQuantity = -int64(stockQuantity)
		transactionSummary.Price = stockPriceFloat64
		l.Debugf("Effectively the first time user is short selling these stocks. No tax is added, but database is going to be updated - %+v", transactionSummary)
	}

	taxTransaction := GetTransactionRef(userId, stockId, TaxTransaction, 0, 0, 0, 0, -int64(tax))

	if tax == 0 {
		return transactionSummary, nil
	}

	return transactionSummary, taxTransaction
}

func PerformBuyFromExchangeTransaction(userId uint32, stockId uint32, stockQuantity uint64) (*Transaction, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":              "PerformBuyFromExchangeTransaction",
		"param_userId":        userId,
		"param_stockId":       stockId,
		"param_stockQuantity": stockQuantity,
	})

	l.Infof("PerformBuyFromExchangeTransaction requested")

	if isBankrupt := IsStockBankrupt(stockId); isBankrupt {
		l.Infof("Stock already bankrupt. Returning function.")
		return nil, StockBankruptError{}
	}

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

	transaction := GetTransactionRef(userId, stockId, FromExchangeTransaction, 0, int64(stockQuantityRemoved), price, 0, -int64(price*stockQuantityRemoved))

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

	// Tax Calculation
	transactionSummary, taxTransaction := getTaxForBiddingUser(tx, stockId, stockQuantityRemoved, price, user)

	if err := tx.Save(transactionSummary).Error; err != nil {
		return errorHelper("Error updating the transaction summary. Rolling back. Error : +%v", err)
	}

	l.Debugf("TransactionSummary table updated successfully.")

	if err := tx.Save(transaction).Error; err != nil {
		return errorHelper("Error creating the transaction. Rolling back. Error: %+v", err)
	}

	l.Debugf("Added transaction to Transactions table")

	//save taxTransaction
	if taxTransaction != nil {
		if err := tx.Save(taxTransaction).Error; err != nil {
			return errorHelper("Error creating the TaxTransaction - %+v - for asking user in the db. Rolling back. Error : +%v", taxTransaction, err)
		}
		l.Debugf("Added TaxTransaction to Transactions table.")
	}

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
		if taxTransaction != nil {
			transactionsStream.SendTransaction(taxTransaction.ToProto())
		}

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
		l.Infof("Done. One of the orders already closed. %d and %d", askStatus, bidStatus)
		return askStatus, bidStatus, nil
	}

	/* We're here, so both orders are open now */

	var updateDataStreams = func(askTrans, bidTrans, askTaxTrans, bidTaxTrans *Transaction) {
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

		if askTaxTrans != nil {
			transactionsStream.SendTransaction(askTaxTrans.ToProto())
		}

		if bidTaxTrans != nil {
			transactionsStream.SendTransaction(bidTaxTrans.ToProto())
		}

		l.Infof("Sent through the datastreams")
	}

	// Begin transaction
	db := getDB()
	tx := db.Begin()

	//Check if bidder has enough cash
	reservedCashForOrder, _, err := getPlaceOrderTransactionDetails(bid.Id, false) // Total cash reserved for whole order
	if err != nil {
		l.Errorf("Error while returning reserved cash. Error: %+v", err)
		return AskUndone, BidUndone, nil
	}
	reservedCashForTrade := int64(float64(reservedCashForOrder) * float64(stockTradeQty) / float64(bid.StockQuantity)) // Part of total allowed for stockTradeQty
	total := int64(stockTradePrice * stockTradeQty)                                                                    // Cash required for the Ask order
	cashLeft := int64(biddingUser.Cash) - total + reservedCashForTrade

	//User has enough stocks reserved. Use that to make transaction
	askTransaction := GetTransactionRef(ask.UserId, ask.StockId, OrderFillTransaction, -int64(stockTradeQty), 0, stockTradePrice, 0, total)
	bidTransaction := GetTransactionRef(bid.UserId, bid.StockId, OrderFillTransaction, 0, int64(stockTradeQty), stockTradePrice, -reservedCashForTrade, -total+reservedCashForTrade)

	// save old cash for rolling back
	askingUserOldCash := askingUser.Cash
	biddingUserOldCash := biddingUser.Cash
	biddingUserOldReservedCash := biddingUser.ReservedCash

	//calculate user's updated cash
	askingUser.Cash += uint64(stockTradeQty) * stockTradePrice

	// bidding user's cash will first be subtracted from reserved cash and then if required from bidding user's cash
	biddingUser.Cash = uint64(cashLeft)
	biddingUser.ReservedCash = uint64(math.Max(0, float64(int64(biddingUser.ReservedCash)-reservedCashForTrade)))

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

	var revertToOldState = func(fmt string, willRollBack bool, args ...interface{}) {
		l.Errorf(fmt, args...)
		askingUser.Cash = askingUserOldCash
		biddingUser.Cash = biddingUserOldCash
		biddingUser.ReservedCash = biddingUserOldReservedCash

		ask.StockQuantityFulfilled = oldAskStockQuantityFulfilled
		bid.StockQuantityFulfilled = oldBidStockQuantityFulfilled

		ask.IsClosed = oldAskIsClosed
		bid.IsClosed = oldBidIsClosed

		if willRollBack {
			tx.Rollback()
		}
	}

	if cashLeft < MINIMUM_CASH_LIMIT {
		revertToOldState("Check1: Failed. Not enough cash. Returning reserved cash back to user.", false, bid.UserId)
		if err := bid.Close(tx); err != nil {
			revertToOldState("Unable to close bid. Rolling back. Error: %+v", true, err)
			return AskUndone, BidUndone, nil
		}
		if err := saveBidCancelOrderTransaction(bid, biddingUser, tx); err != nil {
			revertToOldState("Error saving BidCancelOrderTransaction. Rolling back. Error: %+v", true, err)
			return AskUndone, BidUndone, nil
		}
		if err := tx.Commit().Error; err != nil {
			revertToOldState("Error while commiting. Rolling back. Error: %+v", true, err)
			return AskUndone, BidUndone, nil
		}
		go updateDataStreams(nil, nil, nil, nil)
		go SendNotification(biddingUser.Id, fmt.Sprintf("Your Buy order#%d has been closed due to insufficient cash", bid.Id), false)
		return AskUndone, BidDone, nil
	}

	// Tax Calculation for asking user
	transactionSummary, askTaxTransaction := getTaxForAskingUser(tx, ask.StockId, stockTradeQty, stockTradePrice, askingUser)

	// Update transaction summary table for asking user
	if err := tx.Save(transactionSummary).Error; err != nil {
		revertToOldState("Error updating the transaction summary for asking user. Rolling back. Error : +%v", true, err)
		return AskUndone, BidUndone, nil

	}
	l.Debugf("TransactionSummary table for asking user updated successfully.")

	// Calculate tax for bidding user
	transactionSummary, bidTaxTransaction := getTaxForBiddingUser(tx, bid.StockId, stockTradeQty, stockTradePrice, biddingUser)

	// Update transaction summary table for bidding user
	if err := tx.Save(transactionSummary).Error; err != nil {
		revertToOldState("Error updating the transaction summary for bidding user. Rolling back. Error : +%v", true, err)
		return AskUndone, BidUndone, nil
	}
	l.Debugf("TransactionSummary table updated successfully for bidding user.")

	//save askTransaction
	if err := tx.Save(askTransaction).Error; err != nil {
		revertToOldState("Error creating the askTransaction. Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}
	l.Debugf("Added askTransaction to Transactions table")

	//save bidTransaction
	if err := tx.Save(bidTransaction).Error; err != nil {
		revertToOldState("Error creating the bidTransaction. Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}
	l.Debugf("Added bidTransaction to Transactions table")

	//save askTaxTransaction
	if askTaxTransaction != nil {
		if err := tx.Save(askTaxTransaction).Error; err != nil {
			revertToOldState("Error creating the askTaxTransaction - %+v. Rolling back. Error : +%v", true, askTaxTransaction, err)
			return AskUndone, BidUndone, nil
		}
		l.Debugf("Added askTaxTransaction to Transactions table.")
	}

	//save bidTaxTransaction
	if bidTaxTransaction != nil {
		if err := tx.Save(bidTaxTransaction).Error; err != nil {
			revertToOldState("Error creating the bidTaxTransaction - %+v. Rolling back. Error : +%v", true, bidTaxTransaction, err)
			return AskUndone, BidUndone, nil
		}
		l.Debugf("Added bidTaxTransaction to Transactions table.")
	}

	//update askingUser
	if err := tx.Save(askingUser).Error; err != nil {
		revertToOldState("Error updating askingUser.Cash Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}

	//update biddingUserCash
	if err := tx.Save(biddingUser).Error; err != nil {
		revertToOldState("Error updating biddingUser.Cash Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}

	//update StockQuantityFulfilled and IsClosed for ask order
	if err := tx.Save(ask).Error; err != nil {
		revertToOldState("Error updating ask.{StockQuantityFulfilled,IsClosed}. Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}

	//update StockQuantityFulfilled and IsClosed for bid order
	if err := tx.Save(bid).Error; err != nil {
		revertToOldState("Error updating bid.{StockQuantityFulfilled,IsClosed}. Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}

	// insert an OrderFill
	of := &OrderFill{
		AskId:         ask.Id,
		BidId:         bid.Id,
		TransactionId: askTransaction.Id, // We'll always store Ask
	}
	if err := tx.Save(of).Error; err != nil {
		revertToOldState("Error saving an orderfill. Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}

	//Commit transaction
	if err := tx.Commit().Error; err != nil {
		revertToOldState("Error comming transaction. Rolling back. Error: %+v", true, err)
		return AskUndone, BidUndone, nil
	}

	go updateDataStreams(askTransaction, bidTransaction, askTaxTransaction, bidTaxTransaction)

	UpdateStockVolume(ask.StockId, stockTradeQty)
	l.Infof("Transaction committed successfully. Traded %d at %d per stock. Total %d.", stockTradeQty, stockTradePrice, total)

	if err := UpdateStockPrice(ask.StockId, stockTradePrice, stockTradeQty); err != nil {
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

	// HACK. The line below is a hack to ensure that nothing breaks in existing code.
	// Previously, askTransaction would have negative stockTradeQty which would get sent to the datastream
	// Now, it's 0 as user doesn't actually have to lose stocks anymore. Re-setting the object just before sending back the tranasction
	// ensures that the rest of the matching engine / order book code functions without requiring changes. It's not ideal but I think it'll
	// do for now.
	askTransaction.StockQuantity = -int64(stockTradeQty)
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
	if isBankrupt := IsStockBankrupt(stockId); isBankrupt {
		l.Infof("Stock already bankrupt. Returning function.")
		return nil, StockBankruptError{}
	}

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
	mortgagePrice := allStocks.m[stockId].stock.CurrentPrice
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

	transaction := GetTransactionRef(userId, stockId, MortgageTransaction, 0, stockQuantity, mortgagePrice, 0, trTotal)

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

func GetReservedStocksOwned(userId uint32) (map[uint32]int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetReservedStocksOwned",
		"userId": userId,
	})

	l.Info("GetReservedStocksOwned requested")

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

	sql := "Select stockId, sum(reservedStockQuantity) as reservedStockQuantity from Transactions where userId=? group by stockId"
	rows, err := db.Raw(sql, userId).Rows()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer rows.Close()

	reservedStocksOwned := make(map[uint32]int64)
	for rows.Next() {
		var stockId uint32
		var reservedStockQty int64
		rows.Scan(&stockId, &reservedStockQty)

		reservedStocksOwned[stockId] = reservedStockQty
	}

	return reservedStocksOwned, nil
}

// savePlaceOrderTransaction saves PlaceOrderTransaction and creates a mapping between orderId and
func savePlaceOrderTransaction(orderID uint32, placeOrderTransaction *Transaction, isAsk bool, tx *gorm.DB) error {
	if err := tx.Save(placeOrderTransaction).Error; err != nil {
		return err
	}

	orderDepositTransaction := makeOrderDepositTransactionRef(
		placeOrderTransaction.Id,
		orderID,
		isAsk,
	)

	if err := tx.Save(orderDepositTransaction).Error; err != nil {
		return err
	}

	return nil
}

// Logout removes the user from RAM.
func Logout(userID uint32) {
	userLocks.Lock()
	delete(userLocks.m, userID)
	userLocks.Unlock()
}

func IsUserPhoneVerified(userId uint32) bool {
	userLocks.m[userId].RLock()
	defer userLocks.m[userId].RUnlock()
	return userLocks.m[userId].user.IsPhoneVerified
}

func IsAdminAuth(userId uint32) bool {
	userLocks.m[userId].RLock()
	defer userLocks.m[userId].RUnlock()
	return userLocks.m[userId].user.IsAdmin
}

func IsUserOTPBlocked(userId uint32) bool {
	userLocks.m[userId].RLock()
	defer userLocks.m[userId].RUnlock()
	return userLocks.m[userId].user.IsOTPBlocked
}

func GetUserOTPRequestCount(userId uint32) int64 {
	userLocks.m[userId].RLock()
	defer userLocks.m[userId].RUnlock()
	return userLocks.m[userId].user.OTPRequestCount
}

func IsUserBlocked(userId uint32) bool {
	userLocks.m[userId].RLock()
	defer userLocks.m[userId].RUnlock()
	return userLocks.m[userId].user.IsBlocked
}

func GetUserBlockCount(userId uint32) int64 {
	userLocks.m[userId].RLock()
	defer userLocks.m[userId].RUnlock()
	return userLocks.m[userId].user.BlockCount
}

func SetBlockUser(userId uint32, isBlocked bool, penalty uint64) error {
	var l = logger.WithFields(logrus.Fields{
		"method":          "SetBlockUser",
		"param_userId":    userId,
		"param_isBlocked": isBlocked,
	})
	l.Debugf("Attempting to setBlock for users")

	db := getDB()

	ch, user, err := getUserExclusively(userId)
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	if err == UserNotFoundError {
		return UserNotFoundError
	} else if err != nil {
		return InternalServerError
	}

	if user.IsBlocked == isBlocked {
		return nil
	}
	oldIsBlocked := user.IsBlocked
	oldBlockCount := user.BlockCount
	user.IsBlocked = isBlocked

	if isBlocked {
		user.BlockCount = user.BlockCount + 1
	} else {
		if user.BlockCount > 0 {
			user.BlockCount = user.BlockCount - 1
		} else {
			user.BlockCount = 0
		}
	}

	oldCash := user.Cash

	//Penalty added while blocking user
	user.Cash -= penalty

	if err := db.Save(&user).Error; err != nil {
		l.Errorf("Error saving user. Failing. %+v", err)
		user.IsBlocked = oldIsBlocked
		user.BlockCount = oldBlockCount
		user.Cash = oldCash
		return InternalServerError
	}

	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: userId,
		Ub: &UserBlockState{
			IsBlocked: isBlocked,
			Cash:      user.Cash,
		},
		GsType: UserBlockStateUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	SendPushNotification(userId, PushNotification{
		Title:   "Message from Dalal Street!",
		Message: "Your account has been blocked for violating the game's code of conduct and a penalty has been deducted from your cash, visit the site to appeal the ban.",
		LogoUrl: fmt.Sprintf("%v/static/dalalfavicon.png", config.BackendUrl),
	})

	return nil

}

func UnBlockAllUsers() error {
	var l = logger.WithFields(logrus.Fields{
		"method": "UnBlockAllUsers",
	})
	l.Debugf("Attempting to unblock all users")

	db := getDB()
	allUserIds := []uint32{}
	db.Table("Users").Pluck("id", &allUserIds)

	for _, userId := range allUserIds {
		ch, user, err := getUserExclusively(userId)
		l.Debugf("Acquired %v", userId)

		if err != nil {
			close(ch)
			return InternalServerError
		}

		gameStateStream := datastreamsManager.GetGameStateStream()
		if user.IsBlocked && user.BlockCount < int64(config.MaxBlockCount) {
			user.IsBlocked = false
			if err := db.Save(&user).Error; err != nil {
				l.Errorf("Error saving user. Failing. %+v", err)
				user.IsBlocked = true
				close(ch)
				return InternalServerError
			}

			g := &GameState{
				UserID: userId,
				Ub: &UserBlockState{
					IsBlocked: false,
				},
				GsType: UserBlockStateUpdate,
			}
			gameStateStream.SendGameStateUpdate(g.ToProto())
			l.Debugf("Unblocked %v", userId)
		}
		close(ch)
		l.Debugf("Released exclusive write on userId %v", userId)
	}

	return nil
}

//GetUserStockWorth returns total stockworth of a user including reserved StockWorth
func GetUserStockWorth(userId uint32) (int64, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "GetUserStockWorth",
		"user_id": userId,
	})

	l.Debugf("Attempting to get stockworth")

	var stockWorth int64
	ch, user, err := getUserExclusively(userId)

	if err != nil {
		close(ch)
		return stockWorth, nil
	}
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	stockWorth = user.Total - (int64(user.Cash) + int64(user.ReservedCash))

	l.Debugf("Got %d \n", stockWorth)

	return stockWorth, nil
}
