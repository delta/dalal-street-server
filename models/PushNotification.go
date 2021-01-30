package models

import (
	"bytes"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
)

type UserSubscription struct {
	ID uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserID uint32 `gorm:"column:userId; not null" json:"user_id"`
	EndPoint string `gorm:"column:endpoint; not null" json:"end_point"`
	P256dh string `gorm:"column:p256dh; not null" json:"p256dh"`
	Auth string `gorm:"column:auth; not null" json:"auth"`
}


func (UserSubscription) TableName() string {
	return "UserSubscription"
}

// Adds the subscription keys for sending push notifications
func AddUserSubscription(email string ,data string) error {

	var l = logger.WithFields(logrus.Fields{
		"method":      "Add User Subscription",
		"param_email": email,
		"param_data":  data,
	})

	l.Infof("Add User subscription details requsted")

	db := getDB()

	user := User{
		Email : email,
	}

	err := db.Table("Users").Where("email = ?",email).First(&user).Error; 

	if err != nil {
	   l.Errorf("User not found in Database")
	   return UserNotFoundError
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(data),&result)

	keys := result["keys"].(map[string]interface{})

	userSubscription := &UserSubscription{
		UserID :   user.Id,
		EndPoint : result["endpoint"].(string),
		P256dh : keys["p256dh"].(string),
		Auth : keys["auth"].(string),
	}   

	if err := db.Table("UserSubscription").Save(userSubscription).Error; err != nil {
		l.Errorf("Error saving User Subscription. Failing. %+v",err)
		return InternalServerError
	}

	return nil
}

/** WEB PUSH LOGIC */

func sendNotification(s *UserSubscription) {
	curve := elliptic.P256()

	// get dh key for creating a common key
	dh, err := decodeSubscriptionKey(s.P256dh)
	if err != nil {
		// handle error
	}
	clientAuthSecret, err := decodeSubscriptionKey(s.Auth)

	serverPrivateKey, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	serverPublicKey := elliptic.Marshal(curve, x, y)

	// create a common key with dh
	sharedX, sharedY := elliptic.Unmarshal(curve, dh)

	if sharedX == nil {
		// error while generating shared key
		// public key is not a point on curve
		// handle error
	}
	sx, _ := curve.ScalarMult(sharedX, sharedY, serverPrivateKey)
	sharedSecret := sx.Bytes()

	// HKDF
	// pseudo-random-key
	salt := generateSalt()
	contentEncryptionKeyInfo := bytes.NewBuffer([]byte("Content-Encoding: aesgcm\x00")) // ends with null, 
	contentEncryptionKeyInfo.Write(dh)
	contentEncryptionKeyInfo.Write(serverPublicKey)
	

	prk := HKDF(clientAuthSecret, sharedSecret, []byte("Content-Encoding: auth\x00"), 32)
	contentEncryptionKey := HKDF(salt, prk, contentEncryptionKeyInfo.Bytes(), 32)

	nonceInfo := bytes.NewBuffer([]byte("Content-Encoding: nonce\x00")) // ends with null, 
	nonceInfo.Write(dh)
	nonceInfo.Write(serverPublicKey)
	nonce := HKDF(salt, prk, nonceInfo.Bytes(), 12)

}

// helper functions

func decodeSubscriptionKey(key string) ([]byte, error) {
	buf := bytes.NewBufferString(key)
	if rem := len(key) % 4; rem != 0 {
		// add padding
		buf.WriteString(strings.Repeat("=", 4-rem))
	}

	return base64.URLEncoding.DecodeString(buf.String())
}

func HKDF(salt, ikm, info  []byte, length int) []byte {
	
	if(length > 32){
		// handle error
	}
	
	keyHmac := hmac.New(sha256.New, salt)
	key := keyHmac.Sum(ikm)

	infoHmac := hmac.New(sha256.New, key)
	infoHmac.Write(info)
	infoHmac.Write([]byte("\x01"))

	return infoHmac.Sum(nil)[:length]
}

func generateSalt() ([]byte) {
	salt := make([]byte, 16)
	rand.Read(salt)
	return salt
}