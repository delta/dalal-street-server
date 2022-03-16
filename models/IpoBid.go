package models

import (
	"fmt"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/sirupsen/logrus"

	"github.com/delta/dalal-street-server/utils"
)

type IpoBid struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId       uint32 `gorm:"column:userId;not null" json:"user_id"`
	IpoStockId   uint32 `gorm:"column:ipoStockId;not null" json:"ipo_stock_id"`
	SlotPrice    uint64 `gorm:"column:slotPrice;not null" json:"slot_price"`
	SlotQuantity uint32 `gorm:"column:slotQuantity;not null" json:"slot_quantity"`
	IsFulfilled  bool   `gorm:"column:isFulFilled;not null" json:"is_fulfilled"`
	IsClosed     bool   `gorm:"column:isClosed;not null" json:"is_closed"`
	CreatedAt    string `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt    string `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (*IpoBid) TableName() string {
	return "IpoBids"
}

func (ipoBid *IpoBid) ToProto() *models_pb.IpoBid {

	pIpoBid := &models_pb.IpoBid{
		Id:           ipoBid.Id,
		UserId:       ipoBid.UserId,
		IpoStockId:   ipoBid.IpoStockId,
		SlotPrice:    ipoBid.SlotPrice,
		SlotQuantity: ipoBid.SlotQuantity,
		IsFulfilled:  ipoBid.IsFulfilled,
		IsClosed:     ipoBid.IsClosed,
		CreatedAt:    ipoBid.CreatedAt,
		UpdatedAt:    ipoBid.UpdatedAt,
	}

	return pIpoBid
}

type IpoOrderStockLimitExceeded struct{}

func (e IpoOrderStockLimitExceeded) Error() string {
	return "A user can only bid for a maximum of 1 IPO slot"
}

func CreateIpoBid(UserId uint32, IpoStockId uint32, SlotQuantity uint32) (uint32, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":             "CreateIpoBid",
		"param_userId":       fmt.Sprintf("%+v", UserId),
		"param_ipoStockId":   fmt.Sprintf("%+v", IpoStockId),
		"param_slotQuantity": fmt.Sprintf("%+v", SlotQuantity),
	})

	l.Debugf("Attempting to create ipoBid")

	if SlotQuantity != 1 {
		return 0, IpoOrderStockLimitExceeded{}
	}

	db := getDB()

	IpoStock := &IpoStock{}
	db.First(IpoStock, IpoStockId)

	ch, user, err := getUserExclusively(UserId)

	if err != nil {
		l.Errorf("Error fetching user form db %+v", err)
		return 0, err
	}

	if user.Cash < IpoStock.SlotPrice {
		return 0, NotEnoughCashError{}
	}

	close(ch)

	if !IpoStock.IsBiddable {
		return 0, IpoNotBiddableError{IpoStockId}
	}

	OldIpoBid := &IpoBid{}
	if !db.Where("userId = ? AND isClosed = ? AND ipoStockId = ?", UserId, false, IpoStockId).First(&OldIpoBid).RecordNotFound() {
		return 0, IpoOrderStockLimitExceeded{}
	}

	NewIpoBid := &IpoBid{
		UserId:       UserId,
		IpoStockId:   IpoStockId,
		SlotPrice:    IpoStock.SlotPrice,
		SlotQuantity: SlotQuantity,
		CreatedAt:    utils.GetCurrentTimeISO8601(),
		IsFulfilled:  false,
		IsClosed:     false,
	}
	NewIpoBid.UpdatedAt = NewIpoBid.CreatedAt

	price := uint64(SlotQuantity) * IpoStock.SlotPrice
	// add ipo transaction in transaction table
	if err := SaveNewIpoBidTransaction(NewIpoBid, UserId, price); err != nil {
		l.Error(err)
		return 0, err
	}

	l.Debugf("Created ipoBid. Id: %d", NewIpoBid.Id)

	return NewIpoBid.Id, nil
}

func CancelIpoBid(IpoBidId uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":         "IpoBid.Cancel",
		"param_ipoBidId": fmt.Sprintf("%+v", IpoBidId),
	})

	l.Debugf("Attempting")

	db := getDB()

	IpoBidToCancel := &IpoBid{}

	if err := db.First(&IpoBidToCancel, IpoBidId).Error; err != nil {
		return err
	}

	if IpoBidToCancel.IsClosed {
		return AlreadyClosedError{IpoBidToCancel.Id}
	}

	IpoStock1 := &IpoStock{}
	db.First(IpoStock1, IpoBidToCancel.IpoStockId)

	if !IpoStock1.IsBiddable {
		return IpoNotBiddableError{IpoBidToCancel.IpoStockId}
	}

	IpoBidToCancel.IsClosed = true
	IpoBidToCancel.UpdatedAt = utils.GetCurrentTimeISO8601()

	price := uint64(IpoBidToCancel.SlotQuantity) * IpoBidToCancel.SlotPrice

	if err := SaveCancelledIpoBidTransaction(IpoBidToCancel, price); err != nil {
		l.Error(err)
		return err
	}

	l.Debugf("Done")
	return nil
}

func GetMyIpoBids(userId uint32) ([]*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyIpoBids",
		"userId": userId,
	})

	l.Infof("Attempting to get ipoBid orders for userId : %v", userId)

	db := getDB()

	var myIpoBids []*IpoBid

	if err := db.Where("userId = ?", userId).Find(&myIpoBids).Error; err != nil {
		l.Error(err)
		return nil, err
	}

	l.Infof("Successfully fetched ipoBid orders for userId : %v", userId)
	return myIpoBids, nil
}

func SaveNewIpoBidTransaction(NewIpoBid *IpoBid, UserId uint32, price uint64) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "SaveNewIpoBidTransaction",
		"IpoStockId": NewIpoBid.Id,
	})

	db := getDB()
	tx := db.Begin()

	ch, BiddingUser, err := getUserExclusively(UserId)
	l.Debugf("Acquired")

	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	if err != nil {
		l.Errorf("Error acquiring user. Failing. %+v", err)
		return InternalServerError
	}

	oldCash := BiddingUser.Cash
	oldReservedCash := BiddingUser.ReservedCash

	BiddingUser.Cash -= price
	BiddingUser.ReservedCash += price

	if err := tx.Create(NewIpoBid).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error creating ipo bid %+v", err)
		tx.Rollback()
		return err
	}

	if err := tx.Save(&BiddingUser).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error saving user %+v", err)
		tx.Rollback()
		return err
	}

	if err := tx.Exec("INSERT INTO Transactions (UserId, StockId, Type, ReservedStockQuantity, StockQuantity, Price, ReservedCashTotal, Total, CreatedAt) VALUES (?, NULL, 10, 0, 0, 0, ?, ?, ?);", BiddingUser.Id, int64(price), -int64(price), utils.GetCurrentTimeISO8601()).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error saving transaction %+v", err)
		tx.Rollback()
		return err
	}

	PlaceIpoBidTransaction := &Transaction{}

	if err := tx.First(&PlaceIpoBidTransaction, "userId = ? AND type = ? AND total = ?", BiddingUser.Id, IpoAllotmentTransaction, -int64(price)).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error retreivng transaction %+v", err)
		tx.Rollback()
		return err
	}

	transactionsStream := datastreamsManager.GetTransactionsStream()
	transactionsStream.SendTransaction(PlaceIpoBidTransaction.ToProto())

	if err := tx.Commit().Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error committing transaction %+v", err)
		tx.Rollback()
		return err
	}

	return nil
}

func SaveCancelledIpoBidTransaction(IpoBidToCancel *IpoBid, price uint64) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "SaveCancelledIpoBidTransaction",
		"IpoStockId": IpoBidToCancel.Id,
	})

	db := getDB()
	tx := db.Begin()

	ch, BiddingUser, err := getUserExclusively(IpoBidToCancel.UserId)
	l.Debugf("Acquired")

	defer func() {
		close(ch)
		l.Debugf("Released exclusive write on user")
	}()

	if err != nil {
		l.Errorf("Error acquiring user. Failing. %+v", err)
		return InternalServerError
	}

	oldCash := BiddingUser.Cash
	oldReservedCash := BiddingUser.ReservedCash

	BiddingUser.Cash += price
	BiddingUser.ReservedCash -= price

	if err := tx.Save(IpoBidToCancel).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error cancelling ipo bid %+v", err)
		tx.Rollback()
		return err
	}

	if err := tx.Save(&BiddingUser).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error cancelling ipo bid %+v", err)
		tx.Rollback()
		return err
	}

	if err := tx.Exec("INSERT INTO Transactions (UserId, StockId, Type, ReservedStockQuantity, StockQuantity, Price, ReservedCashTotal, Total, CreatedAt) VALUES (?, NULL, 10, 0, 0, 0, ?, ?, ?);", BiddingUser.Id, -int64(price), int64(price), utils.GetCurrentTimeISO8601()).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error saving transaction %+v", err)
		tx.Rollback()
		return err
	}

	CancelIpoBidTransaction := &Transaction{}

	if err := tx.First(&CancelIpoBidTransaction, "userId = ? AND type = ? AND total = ?", BiddingUser.Id, IpoAllotmentTransaction, -int64(price)).Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error retreivng transaction %+v", err)
		tx.Rollback()
		return err
	}

	transactionsStream := datastreamsManager.GetTransactionsStream()
	transactionsStream.SendTransaction(CancelIpoBidTransaction.ToProto())

	if err := tx.Commit().Error; err != nil {
		BiddingUser.Cash = oldCash
		BiddingUser.ReservedCash = oldReservedCash
		l.Errorf("Error committing transaction %+v", err)
		tx.Rollback()
		return err
	}

	return nil
}
