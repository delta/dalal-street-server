package models

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"errors"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/hkdf"

	"github.com/sirupsen/logrus"
)

// ErrMaxPadExceeded if length of marshalled payload is more than the max length
var ErrMaxPadExceeded = errors.New("payload has exceeded the maximum length")

// UserSubscription is stores all the subscription info along with the user foreign key
type UserSubscription struct {
	ID       uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserID   uint32 `gorm:"column:userId; not null" json:"user_id"`
	EndPoint string `gorm:"column:endpoint; not null" json:"end_point"`
	P256dh   string `gorm:"column:p256dh; not null" json:"p256dh"`
	Auth     string `gorm:"column:auth; not null" json:"auth"`
}

// PushNotification the message format for the notification
type PushNotification struct {
	Title       string
	Message     string
	LogoUrl     string
	FrontEndUrl string
}

// TableName returns UserSubscription table name
func (UserSubscription) TableName() string {
	return "UserSubscription"
}

// AddUserSubscription Adds the subscription keys for sending push notifications
func AddUserSubscription(userID uint32, data string) error {

	var l = logger.WithFields(logrus.Fields{
		"method":       "Add User Subscription",
		"param_userid": userID,
		"param_data":   data,
	})

	l.Infof("Add User subscription details requsted")

	db := getDB()

	user := User{
		Id: userID,
	}

	err := db.Table("Users").Where("id = ?", userID).First(&user).Error

	if err != nil {
		l.Errorf("User not found in Database")
		return UserNotFoundError
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(data), &result)

	keys := result["keys"].(map[string]interface{})

	userSubscription := &UserSubscription{
		UserID:   userID,
		EndPoint: result["endpoint"].(string),
		P256dh:   keys["p256dh"].(string),
		Auth:     keys["auth"].(string),
	}

	if err := db.Table("UserSubscription").Save(userSubscription).Error; err != nil {
		l.Errorf("Error saving User Subscription. Failing. %+v", err)
		return InternalServerError
	}

	return nil
}

// SendPushNotification sends notification to the user
func SendPushNotification(userID uint32, p PushNotification) error {

	var l = logger.WithFields(logrus.Fields{
		"method":        "Sending push notification",
		"param_data":    p,
		"param_user_id": userID,
	})

	l.Infof("Sending push notifications to the users")

	db := getDB()
	p.FrontEndUrl = config.FrontEndUrl

	var subscriptions []UserSubscription

	if userID == 0 {
		// broadcast notif
		l.Infof("A broadcast notification was requested")
		if err := db.Table("UserSubscription").Find(&subscriptions).Error; err != nil {
			l.Errorf("Error while finding the user subscription, %+v", err)
			return err
		}
		l.Debugf("Found a total of %v subscriptions were found", len(subscriptions))
	} else {
		// single notif
		l.Infof("Notification for a specific user was requested")
		if err := db.Table("UserSubscription").Where("userId = ?", userID).Find(&subscriptions).Error; err != nil {
			l.Errorf("Error while finding the user subscription, %+v", err)
			return err
		}
		l.Debugf("Found a total of %v subscriptions for the user", len(subscriptions))
	}

	for i, sub := range subscriptions {
		l.Debugf("Sending notif to the %v-th subscription, %+v", i, sub)
		message, err := json.Marshal(p)
		if err != nil {
			l.Errorf("Error while marshalling payload, %+v . Error, %+v", p, err)
		}
		resp, err := sendPushNotification(message, &sub, &options{
			Subscriber:      config.PushNotificationEmail,
			VAPIDPublicKey:  config.PushNotificationVAPIDPublicKey,
			VAPIDPrivateKey: config.PushNotificationVAPIDPrivateKey,
		})
		if err != nil {
			l.Errorf("Couldn't send notification to the subscription, %+v. Error : %+v", sub, err)
		}
		defer resp.Body.Close()
	}
	l.Infof("Successfully sent push notification to the user")

	return nil
}

/**
WEB PUSH LOGIC
*/

const maxRecordSize uint32 = 4096

// saltFunc generates a salt of 16 bytes
var saltFunc = func() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return salt, err
	}

	return salt, nil
}

type options struct {
	RecordSize      uint32 // Limit the record size, (optional)
	Subscriber      string // Sub in VAPID JWT token
	VAPIDPublicKey  string // VAPID public key, passed in VAPID Authorization header
	VAPIDPrivateKey string // VAPID private key, used to sign VAPID JWT token
}

func sendPushNotification(message []byte, s *UserSubscription, options *options) (*http.Response, error) {
	// Authentication secret (auth_secret)
	authSecret, err := decodeSubscriptionKey(s.Auth)
	if err != nil {
		return nil, err
	}

	// dh (Diffie Hellman)
	dh, err := decodeSubscriptionKey(s.P256dh)
	if err != nil {
		return nil, err
	}

	// Generate 16 byte salt
	salt, err := saltFunc()
	if err != nil {
		return nil, err
	}

	// Create the ecdh_secret shared key pair
	curve := elliptic.P256()

	// Application server key pairs (single use)
	localPrivateKey, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	localPublicKey := elliptic.Marshal(curve, x, y)

	// Combine application keys with dh
	sharedX, sharedY := elliptic.Unmarshal(curve, dh)
	if sharedX == nil {
		return nil, errors.New("Unmarshal Error: Public key is not a valid point on the curve")
	}

	sx, _ := curve.ScalarMult(sharedX, sharedY, localPrivateKey)
	sharedECDHSecret := sx.Bytes()

	hash := sha256.New

	// ikm
	prkInfoBuf := bytes.NewBuffer([]byte("WebPush: info\x00"))
	prkInfoBuf.Write(dh)
	prkInfoBuf.Write(localPublicKey)

	prkHKDF := hkdf.New(hash, sharedECDHSecret, authSecret, prkInfoBuf.Bytes())
	ikm, err := getHKDFKey(prkHKDF, 32)
	if err != nil {
		return nil, err
	}

	// Derive Content Encryption Key
	contentEncryptionKeyInfo := []byte("Content-Encoding: aes128gcm\x00")
	contentHKDF := hkdf.New(hash, ikm, salt, contentEncryptionKeyInfo)
	contentEncryptionKey, err := getHKDFKey(contentHKDF, 16)
	if err != nil {
		return nil, err
	}

	// Derive the Nonce
	nonceInfo := []byte("Content-Encoding: nonce\x00")
	nonceHKDF := hkdf.New(hash, ikm, salt, nonceInfo)
	nonce, err := getHKDFKey(nonceHKDF, 12)
	if err != nil {
		return nil, err
	}

	// Cipher
	c, err := aes.NewCipher(contentEncryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	// Get the record size
	recordSize := options.RecordSize
	if recordSize == 0 {
		recordSize = maxRecordSize
	}

	recordLength := int(recordSize) - 16

	// Encryption Content-Coding Header
	recordBuf := bytes.NewBuffer(salt)

	rs := make([]byte, 4)
	binary.BigEndian.PutUint32(rs, recordSize)

	recordBuf.Write(rs)
	recordBuf.Write([]byte{byte(len(localPublicKey))})
	recordBuf.Write(localPublicKey)

	// Data
	dataBuf := bytes.NewBuffer(message)

	// Pad content to max record size - 16 - header
	// Padding ending delimeter
	dataBuf.Write([]byte("\x02"))
	if err := pad(dataBuf, recordLength-recordBuf.Len()); err != nil {
		return nil, err
	}

	// Compose the ciphertext
	ciphertext := gcm.Seal([]byte{}, nonce, dataBuf.Bytes(), nil)
	recordBuf.Write(ciphertext)

	// POST request
	req, err := http.NewRequest("POST", s.EndPoint, recordBuf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Length", strconv.Itoa(len(ciphertext)))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", strconv.Itoa(3000))

	// Get VAPID Authorization header
	vapidAuthHeader, err := getVAPIDAuthorizationHeader(
		s.EndPoint,
		options.Subscriber,
		options.VAPIDPublicKey,
		options.VAPIDPrivateKey,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", vapidAuthHeader)

	client := &http.Client{}

	return client.Do(req)
}

// decodeSubscriptionKey decodes a base64 subscription key.
// if necessary, add "=" padding to the key for URL decode
func decodeSubscriptionKey(key string) ([]byte, error) {
	// "=" padding
	buf := bytes.NewBufferString(key)
	if rem := len(key) % 4; rem != 0 {
		buf.WriteString(strings.Repeat("=", 4-rem))
	}

	bytes, err := base64.StdEncoding.DecodeString(buf.String())
	if err == nil {
		return bytes, nil
	}

	return base64.URLEncoding.DecodeString(buf.String())
}

// Returns a key of length "length" given an hkdf function
func getHKDFKey(hkdf io.Reader, length int) ([]byte, error) {
	key := make([]byte, length)
	n, err := io.ReadFull(hkdf, key)
	if n != len(key) || err != nil {
		return key, err
	}

	return key, nil
}

func pad(payload *bytes.Buffer, maxPadLen int) error {
	payloadLen := payload.Len()
	if payloadLen > maxPadLen {
		return ErrMaxPadExceeded
	}

	padLen := maxPadLen - payloadLen

	padding := make([]byte, padLen)
	payload.Write(padding)

	return nil
}

// Generates the ECDSA public and private keys for the JWT encryption
func generateVAPIDHeaderKeys(privateKey []byte) *ecdsa.PrivateKey {
	// Public key
	curve := elliptic.P256()
	px, py := curve.ScalarMult(
		curve.Params().Gx,
		curve.Params().Gy,
		privateKey,
	)

	pubKey := ecdsa.PublicKey{
		Curve: curve,
		X:     px,
		Y:     py,
	}

	// Private key
	d := &big.Int{}
	d.SetBytes(privateKey)

	return &ecdsa.PrivateKey{
		PublicKey: pubKey,
		D:         d,
	}
}

// getVAPIDAuthorizationHeader
func getVAPIDAuthorizationHeader(
	endpoint,
	subscriber,
	vapidPublicKey,
	vapidPrivateKey string,
) (string, error) {
	// Create the JWT token
	subURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"aud": fmt.Sprintf("%s://%s", subURL.Scheme, subURL.Host),
		"exp": time.Now().Add(time.Hour * 12).Unix(),
		"sub": fmt.Sprintf("mailto:%s", subscriber),
	})

	// Decode the VAPID private key
	decodedVapidPrivateKey, err := decodeVapidKey(vapidPrivateKey)
	if err != nil {
		return "", err
	}

	privKey := generateVAPIDHeaderKeys(decodedVapidPrivateKey)

	// Sign token with private key
	jwtString, err := token.SignedString(privKey)
	if err != nil {
		return "", err
	}

	// Decode the VAPID public key
	pubKey, err := decodeVapidKey(vapidPublicKey)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"vapid t=%s, k=%s",
		jwtString,
		base64.RawURLEncoding.EncodeToString(pubKey),
	), nil
}

func decodeVapidKey(key string) ([]byte, error) {
	bytes, err := base64.URLEncoding.DecodeString(key)
	if err == nil {
		return bytes, nil
	}

	return base64.RawURLEncoding.DecodeString(key)
}
