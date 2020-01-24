package models

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/templates"
	"github.com/delta/dalal-street-server/utils"
	uuid "github.com/satori/go.uuid"
)

//PasswordChangeRequests -> Defines struct
type PasswordChangeRequests struct {
	ID        uuid.UUID `gorm:"primary_key" json:"id"`
	CreatedAt string    `gorm:"column:createdAt;not null" json:"created_at"`
	UserID    uint32    `gorm:"column:userId;not null" json:"user_id"`
}

//PasswordReset -> Takes in email of user ,validates it and sends password change email to user
func PasswordReset(email string) (string, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":      "ForgotPassword",
		"param_email": email,
	})

	l.Infof("PasswordReset requested")

	db := getDB()

	/* Committing to database */

	var currRegis Registration

	err := db.Table("Registrations").Where("email = ?", email).First(&currRegis).Error

	if err != nil {
		l.Errorf("Error while finding user email", err)
		return "Error while finding user", UnauthorizedError
	}

	if currRegis.IsPragyan == true {
		l.Errorf("Error as pragyan user requesting password change email", err)
		return "You have registered using Pragyan Account. Try changing Password on Pragyan Website", PragyanUserError
	}

	tempPass, _ := uuid.NewV4()

	currReset := &PasswordChangeRequests{
		ID:        tempPass,
		CreatedAt: utils.GetCurrentTimeISO8601(),
		UserID:    currRegis.UserId,
	}

	if config.Stage == "docker" {

		l.Debugf("Sending password reset email to %s", email)

		htmlContent := fmt.Sprintf(`%s
							%s
							%s`, templates.HtmlPasswordResetTemplateHead, tempPass, templates.HtmlPasswordResetTemplateTail)
		plainContent := fmt.Sprintf(templates.PlainPasswordResetTemplate, tempPass)

		err = utils.SendEmail("noreply@dalalstreet.com", "Password Reset", email, plainContent, htmlContent)

		if err != nil {
			l.Errorf("Error while sending password reset email to player %s", err)
			return "Error while sending password reset email to player", err
		}
	}

	err = db.Table("PasswordChangeRequests").Save(currReset).Error

	if err != nil {
		l.Error(err)
		return "Error while finding user", err

	}

	return "OK", nil

}

func ChangePassword(tempPass, newPass, confirmPass string) (string, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "	ChangePassword",
		"param_tempPass": tempPass,
		"param_newPass":  newPass,
	})

	l.Debugf("Attempting to change password")

	var requests PasswordChangeRequests
	var currRegis Registration

	if newPass != confirmPass {
		return "Password Mismatch error", PasswordMismatchError
	}

	db := getDB()
	err := db.Table("PasswordChangeRequests").Where("id = ?", tempPass).First(&requests).Error

	if err != nil {
		l.Error(err)
		return "Invalid temporary password", InvalidTemporaryPasswordError
	}

	lastTime, _ := time.Parse(time.RFC3339, requests.CreatedAt)
	currTime := time.Now()
	diff := currTime.Sub(lastTime)
	comp := int64(8.64e+13)
	timeInt := int64(diff)

	stringId := requests.ID.String()

	if timeInt < comp && tempPass == stringId {

		hashedPass, _ := hashPassword(newPass)

		err = db.Table("Registrations").Where("userId = ?", requests.UserID).Find(&currRegis).Error
		currRegis.Password = hashedPass

		err = db.Table("Registrations").Save(currRegis).Error

		if err != nil {
			l.Error(err)
			return "Unable to update server", err
		}
		return "OK", nil
	} else {
		return "Temporary Password Expired", TemporaryPasswordExpiredError
	}

}
