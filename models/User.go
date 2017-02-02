package models

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"encoding/json"
	"errors"
	"time"
	"strconv"

	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/utils"
)

var (
	UnauthorizedError  = errors.New("Invalid credentials")
	NotRegisteredError = errors.New("Not registered on main site")
	InternalError      = errors.New("Internal server error")
)

// User models the User object.
type User struct {
	Id        uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	Email     string `gorm:"unique;not null"`
	Name      string `gorm:"not null"`
	Cash      uint32 `gorm:"not null"`
	Total     int32  `gorm:"not null"`
	CreatedAt string `gorm:"column:createdAt;not null"`
}

// pragyanUser is the structure returned by Pragyan API
type pragyanUser struct {
	Id   uint32	`json:"user_id"`
	Name string `json:"user_fullname"`
}

// User.TableName() is for letting Gorm know the correct table name.
func (User) TableName() string {
	return "Users"
}

// Login() is used to login an existing user or register a new user
// Registration happens provided Pragyan API verifies the credentials
// and the user doesn't exist in our database.
func Login(email, password string) (*User, error) {
	var l = logger.WithFields(logrus.Fields{
		"_method"       : "Login",
		"param_email"   : email,
		"param_password": password,
	})

	pu, err := postLoginToPragyan(email, password)
	if err != nil {
		return nil, err
	}

	l.Debugf("Trying to get user from database. UserId: %d, Name: %s", pu.Id, pu.Name)

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	u := &User{}
	if result := db.First(u, pu.Id); result.Error != nil {
		if !result.RecordNotFound() {
			l.Errorf("Error in loading user info from database: '%s'", result.Error)
			return nil, result.Error
		}

		l.Infof("User (%d, %s, %s) not found in database. Registering new user", pu.Id, email, pu.Name)

		u, err = createUser(pu, email)
		if err != nil {
			return nil, err
		}
	}

	l.Infof("Found user (%d, %s, %s). Logging him in.", u.Id, u.Email, u.Name)

	return u, nil
}

// createUser() creates a user given his email and name.
func createUser(pu pragyanUser, email string) (*User, error) {
	var l = logger.WithFields(logrus.Fields{
		"_method": "createUser",
		"param_id"    : pu.Id,
		"param_email" : email,
		"param_name"  : pu.Name,
	})

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	u := &User{
		Id   : pu.Id,
		Email: email,
		Name : pu.Name,
		Cash : STARTING_CASH,
		Total: STARTING_CASH,
		CreatedAt: time.Now().String(),
	}
	l.Debugf("Creating user")

	err = db.Save(u).Error

	if err != nil {
		l.Error(err)
		return nil, err
	}

	l.Infof("Created user. UserId: %d", u.Id)
	return u, nil
}

// postLoginToPragyan() is used to make a post request to pragyan and return
// a pragyanUser struct.
func postLoginToPragyan(email, password string) (pragyanUser, error) {
	var l = logger.WithFields(logrus.Fields{
		"_method": "postLoginToPragyan",
		"param_email" : email,
		"param_name"  : password,
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
		StatusCode  int			`json:"status_code"`
		Message     message_t	`json:"message"`
	}{}
	json.Unmarshal(body, &r)

	switch r.StatusCode {
	case 200:
		uid, _ := strconv.ParseUint(r.Message.Id, 10, 32)
		pu := pragyanUser{
			Id: uint32(uid),
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
