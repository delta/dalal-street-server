package models

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/delta/dalal-street-server/utils"

	"github.com/Sirupsen/logrus"
)

var (
	PhoneNoAlreadyTakenError = errors.New("This mobile number is already in use.")
	OTPExpiredError          = errors.New("OTP Expired")
	InvalidPhoneNumberError  = errors.New("Invalid Phone Number")
	OTPMismatchError         = errors.New("OTP Mismatch")
	InternalServerError      = errors.New("Internal Server Error")
	SendSMSError             = errors.New("Error in sending sms")
)

type OTP struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	PhoneNo    string `gorm:"column:phoneNo;unique;not null" json:"phone_no"`
	Otp        uint32 `gorm:"column:otp;not null" json:"otp"`
	IsVerified bool   `gorm:"column:isVerified;not null" json:"is_verified"`
	UpdatedAt  string `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (OTP) TableName() string {
	return "OTP"
}

type UserOtp struct {
	UserId uint32 `gorm:"column:userId; not null"`
	OtpId  uint32 `gorm:"column:otpId; not null"`
}

func (UserOtp) TableName() string {
	return "UserOtp"
}

func VerifyOTP(userId, otpNo uint32, phoneNo string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "VerifyOTP",
		"param_otpNo":   otpNo,
		"param_phoneNo": phoneNo,
	})
	l.Debugf("Attempting to verify otp")

	db := getDB()

	var otp = OTP{}

	err := db.Where("phoneNo = ?", phoneNo).First(&otp).Error

	if err != nil {
		return InvalidPhoneNumberError
	}

	if otpNo != otp.Otp {
		l.Debugf("OTP Mismatch. Expected OTP %v, Given OTP %v.", otp.Otp, otpNo)
		return OTPMismatchError
	}

	l.Debugf("OTP Match. Expected OTP %v, Given OTP %v.", otp.Otp, otpNo)
	lastTime, _ := time.Parse(time.RFC3339, otp.UpdatedAt)
	currTime := time.Now()
	diff := int64(currTime.Sub(lastTime).Seconds())
	timeLimit := int64(60 * 10)

	if diff > timeLimit {
		return OTPExpiredError
	}

	otp.IsVerified = true
	tx := db.Begin()
	if err := tx.Save(otp).Error; err != nil {
		l.Error("Error while modifiying IsVerified for phoneNo %s", otp.PhoneNo)
		tx.Rollback()
		return InternalServerError
	}

	userOtp := UserOtp{}
	userOtp.OtpId = otp.Id
	userOtp.UserId = userId
	if err := tx.Save(userOtp).Error; err != nil {
		l.Error("Error while adding to UserOtp table IsVerified for userId %s and otpId %s", userOtp.UserId, userOtp.OtpId)
		tx.Rollback()
		return InternalServerError
	}

	ch, user, err := getUserExclusively(userId)
	l.Debugf("Acquired")
	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	if err != nil {
		l.Errorf("Error updating otp. Failing. %+v", err)
		return InternalServerError
	}
	user.IsPhoneVerified = true
	if err := tx.Save(user).Error; err != nil {
		l.Errorf("Error saving user data. Rolling back. Error: %+v", err)
		tx.Rollback()
		user.IsPhoneVerified = false
		return InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		l.Error("Error commiting otp update transaction")
		tx.Rollback()
		user.IsPhoneVerified = false
		return InternalServerError
	}

	return nil
}

func SendOTP(userId uint32, phoneNo string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "SendOTP",
		"param_userId":  userId,
		"param_phoneNo": phoneNo,
	})
	l.Debugf("Attempting to send otp")

	db := getDB()

	var otp = OTP{}
	err := db.Where("phoneNo = ?", phoneNo).First(&otp).Error
	if err != nil && gorm.IsRecordNotFoundError(err) == false {
		l.Errorf("Error occurred, %v", err)
		return InternalServerError
	}

	if err == nil && otp.IsVerified == true {
		l.Debugf("Phone number %v already in use.", phoneNo)
		return PhoneNoAlreadyTakenError
	}

	rand.Seed(time.Now().UnixNano())
	otpNo := uint32(rand.Int31n(9999-1000) + 1000)

	otp.Otp = otpNo
	otp.PhoneNo = phoneNo
	otp.IsVerified = false
	otp.UpdatedAt = utils.GetCurrentTimeISO8601()

	if err := db.Save(&otp).Error; err != nil {
		l.Errorf("Error saving otp. Failing. %+v", err)
		return InternalServerError
	}

	otpString := fmt.Sprint(otp.Otp)
	smsContent := fmt.Sprintf("[#] Greetings from Pragyan and Dalal Street. Your One Time Password is %s.\nhSG7XAtjOfM", otpString)
	if err := utils.SendSMS(otp.PhoneNo, smsContent); err != nil {
		l.Errorf("Error sending sms. Failing. %+v", err)
		return SendSMSError
	}

	return nil
}
