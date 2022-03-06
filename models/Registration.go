package models

import (
	"errors"
	"fmt"

	"github.com/delta/dalal-street-server/templates"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

var (
	VerificationKeyNotFoundError = errors.New("Verification key not found")
	TransactionError             = errors.New("Transaction failed")
	AlreadyVerifiedError         = errors.New("Account already verified")
	MaximumEmailCountReached     = errors.New("Cannot send verification email again")
	EmailAlreadyVerified         = errors.New("Email already verified")
)

type Registration struct {
	Id                     uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	Email                  string `gorm:"unique;not null" json:"email"`
	Name                   string `gorm:"column:fullName;not null" json:"full_name"`
	UserName               string `gorm:"column:userName;not null" json:"user_name"`
	UserId                 uint32 `gorm:"column:userId;not null" json:"user_id"`
	IsPragyan              bool   `gorm:"column:isPragyan;not null" json:"is_pragyan"`
	Password               string `gorm:"column:password;not null" json:"password"`
	Country                string `gorm:"not null" json:"country"`
	IsVerified             bool   `gorm:"column:isVerified;not null" json:"is_verified"`
	VerificationKey        string `gorm:"column:verificationKey" json:"verification_key"`
	ReferralCodeID         uint32 `gorm:"column:referralCode;default:NULL" json:"referral_code"`
	VerificationEmailCount uint32 `gorm:"column:verificationEmailCount" json:"verification_email_count"`
}

func (Registration) TableName() string {
	return "Registrations"
}

// VerifyAccount is used to handle logic after clicking the email verification
// link sent after registration
func VerifyAccount(verificationKey string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":                "VerifyAccount",
		"param_verificationKey": verificationKey,
	})
	db := getDB()

	var registeredUser = Registration{
		VerificationKey: verificationKey,
	}

	err := db.Where("verificationKey = ?", verificationKey).First(&registeredUser).Error
	if err != nil {
		l.Errorf("No verification key %s", verificationKey)
		return VerificationKeyNotFoundError
	} else if registeredUser.IsVerified == true {
		l.Infof("Already Verified")
		return AlreadyVerifiedError
	} else {
		registeredUser.IsVerified = true
		if err := db.Save(registeredUser).Error; err != nil {
			l.Errorf("Error while modifiying IsVerified for key %s", verificationKey)
			registeredUser.IsVerified = false
		}
		return nil
	}
}

func ResendVerificationEmail(email string) error {

	var l = logger.WithFields(logrus.Fields{
		"method":      "Resending verification email",
		"param_email": email,
	})
	db := getDB()
	var registration = Registration{
		Email: email,
	}
	if err := db.Table("Registrations").Where("email = ?", email).First(&registration).Error; err != nil {
		l.Errorf("Couldn't find the user in registrations, %v", err)
		return UserNotFoundError
	}

	if registration.IsVerified {
		return EmailAlreadyVerified
	}

	if registration.VerificationEmailCount >= config.MaxVerificationEmailRequestCount {
		return MaximumEmailCountReached
	}

	verificationKey := registration.VerificationKey
	l.Debugf("Sending verification email to %s", email)
	verificationURL := fmt.Sprintf("https://dalal.pragyan.org/api/verify?key=%s", verificationKey)
	htmlContent := fmt.Sprintf(`%s
							%s
							%s`, templates.HtmlEmailVerificationTemplateHead, verificationURL, templates.HtmlEmailVerificationTemplateTail)
	plainContent := fmt.Sprintf(templates.PlainEmailVerificationTemplate, verificationURL)
	if err := utils.SendEmail("noreply@dalal.pragyan.org", "Account Verification", email, plainContent, htmlContent); err != nil {
		l.Errorf("Error while sending verification email to player %s", err)
		return err
	}

	registration.VerificationEmailCount++
	if err := db.Table("Registrations").Save(registration).Error; err != nil {
		l.Errorf("Error while updating registration %v", err)
		return err
	}

	return nil
}
