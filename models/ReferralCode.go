package models

import (
	"github.com/sirupsen/logrus"
	"github.com/delta/dalal-street-server/utils"
	"fmt"
	"github.com/jinzhu/gorm"
)

// ReferralCode for new users
type ReferralCode struct {
	ID uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserID uint32 `gorm:"column:userId; not null" json:"user_id"`
	ReferralCode string `gorm:"column:referralCode;no null;unique;" json:"referral_code"`
}

// TableName is for letting Gorm know the correct table name.
func (ReferralCode) TableName() string  {
	return "ReferralCode"
}

/*
	1. Generate a new Referral Code when the user requests for it
	2. Verify if the Referral Code Given by the user is correct
	3.
*/

// GetReferralCode fetches a user's referral code if its exists,
// Else will generate one and sends it to the user 
func GetReferralCode(email string) (string, error) {
	
	var l = logger.WithFields(logrus.Fields{
		"method":      "Generate Referral Code",
		"param_email": email,
	})

	l.Infof("Referral code of user is requested")
	
	db := getDB()

	var rflCode ReferralCode


	var usr =  User{
		Email : email,
	}
	err := db.Table("Users").Where("email = ?", email).First(&usr).Error;
	if err != nil {
		l.Errorf("User not found in db")
		// User was not founds
		return "User was not found", UserNotFoundError
	}

	l.Debugf("user object %+v", usr)
	
	// User is present in db
	err = db.Table("ReferralCode").Where("userId = ?", usr.Id).First(&rflCode).Error; 
	if err == nil {
		// Referral Code already exists
		return rflCode.ReferralCode, nil
	}


	l.Debugf("Generating new referral-code for the user");
	// user doesn't have a referralcode
	// generating referral code for him

	code := utils.RandString(16)
	code = fmt.Sprintf("%s%d",code, usr.Id) // To make sure the generated referral code is unique

	l.Debugf("New referral-code : %v",code)

	newReferralCode := &ReferralCode{
		UserID : usr.Id,
		ReferralCode : code,
	}

	if err := db.Table("ReferralCode").Save(newReferralCode).Error; err != nil {
		// Something went wrong :( 
		return "Something went wrong while generating a ReferralCode", err
	}

	// returning generated referral code
	return code, nil
}

// VerifyReferralCode checks if the user has entered a valid referral code
func VerifyReferralCode(referralCode string) (uint32, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":      "Verifying Referral Code",
		"param_referralCode": referralCode,
	})

	l.Debugf("Verifying Referral Code")
	db :=getDB()

	var code ReferralCode
	if err := db.Table("ReferralCode").Where("referralCode = ?", referralCode).First(&code).Error; err == gorm.ErrRecordNotFound {
		// referral code doesn't exist in database
		return 0, nil
	} else if err == nil {
		// referral code exists
		return code.ID, nil
	}	else {
		// Someother error occurred
		l.Errorf("Error while verifying referral code %v", err)
		return 0, err
	}

}


// AddExtraCredit Adds extra credit for the users
func AddExtraCredit(userID uint32) (error) {

	var l = logger.WithFields(logrus.Fields{
		"method":      "Adding extra credit",
		"param_registration_id": userID,
	})

	l.Infof("Adding extra credit for new users with referral code")

	db := getDB()

	var reg Registration;

	if err := db.Table("Registrations").Where("userId = ?", userID).First(&reg).Error; err != nil {
		
		// not able to find the user, for some reason
		// shdnt happen but still
		return err

	}
	// user exists

	l.Errorf("The referralCode is %v\n", reg.ReferralCodeID);

	if reg.ReferralCodeID == 0  {
		// user didn't use a referral-code
		l.Infof("User didn't use a referral-code")
		return nil
	}

	var referCode ReferralCode
	if err := db.Table("ReferralCode").Where("id = ?", reg.ReferralCodeID).First(&referCode).Error; err != nil {
		// something went wrong
		l.Errorf("Error while querying for the referral code table. %v\n", err)
		return err;
	}

	done, codeProvider, codeUser, err1 := getUserPairExclusive(referCode.UserID, userID)

	if err1 != nil {
		l.Errorf("Some error, %v", err1)
		return err1;
	}

	// creating transactions for adding to db
	tx := db.Begin()

	defer func() {
    if r := recover(); r != nil {
			l.Errorf("Something went wrong %v", r)
			tx.Rollback()
		}
		close(done)
  }()

	l.Debugf("Adding 2k rs to the code provider %v, with cash %v and code user %v with cash %v.", codeProvider.Email, codeProvider.Cash, codeUser.Email, codeUser.Cash)

	// adding extra cash for the user
	codeProvider.Cash += 2000
	codeUser.Cash += 2000

	if err := tx.Save(&codeProvider).Error; err != nil {
		tx.Rollback()
		l.Errorf("Error while updating user's in-game cash. %v\n", err)
		return err
	}
	if err := tx.Save(&codeUser).Error; err != nil {
		tx.Rollback()
		l.Errorf("Error while updating user's in-game cash. %v\n", err)
		return err
	}

	l.Debug("Successfully added money to the users")

	return tx.Commit().Error
}