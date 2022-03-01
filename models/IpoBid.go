package models

import (
	"fmt"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/delta/dalal-street-server/utils"
)

// refer models/Bid.go
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

var IpoBidsMap = struct {
	m map[uint32]*IpoBid
}{
	make(map[uint32]*IpoBid),
}

type IpoOrderStockLimitExceeded struct{}

func (e IpoOrderStockLimitExceeded) Error() string {
	return fmt.Sprintf("A user can only bid for a maximum of 1 IPO slot")
}

// Is this function even needed?
func getIpoBid(id uint32) (*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "getIpoBid",
		"param_id": id,
	})

	l.Debugf("Attempting")

	/* Check if the ipoBid is there in the map */
	_, ok := IpoBidsMap.m[id]
	if ok {
		l.Debugf("Found ipoBid in IpoBidsMap")
		return IpoBidsMap.m[id], nil
	}

	/* Otherwise load from database */
	l.Debugf("Loading ipoBid from database")
	db := getDB()

	ipoBid := &IpoBid{}
	db.First(ipoBid, id)

	if ipoBid == nil {
		l.Errorf("Attempted to get non-existing IpoBid")
		return nil, fmt.Errorf("IpoBid with id %d does not exist", id)
	}

	IpoBidsMap.m[id] = ipoBid

	l.Debugf("Loaded ipoBid from db: %+v", ipoBid)

	return ipoBid, nil
}

func CreateIpoBid(UserId uint32, IpoStockId uint32, SlotQuantity uint32, SlotPrice uint64) (uint32, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":             "CreateIpoBid",
		"param_userId":       fmt.Sprintf("%+v", UserId),
		"param_ipoStockId":   fmt.Sprintf("%+v", IpoStockId),
		"param_slotQuantity": fmt.Sprintf("%+v", SlotQuantity),
	})

	l.Debugf("Attempting")

	if SlotQuantity != 1 {
		return 0, IpoOrderStockLimitExceeded{}
	}

	db := getDB()

	BiddingUser := &User{}
	db.First(BiddingUser, UserId)

	if BiddingUser.Cash < SlotPrice {
		return 0, NotEnoughCashError{}
	}

	var OldIpoBids []*IpoBid

	if err := db.Where("userId = ? AND isClosed = ?", UserId, false).Find(&OldIpoBids).Error; err != nil {
		return 0, err
	}
	for _, OldIpoBid := range OldIpoBids {
		if OldIpoBid.Id != 0 {
			return 0, OrderStockLimitExceeded{}
		}
	} // if user has already made bid on this stock, return error

	IpoStock := &IpoStock{}
	db.First(IpoStock, IpoStockId)
	if IpoStock.IsBiddable == false {
		return 0, IpoNotBiddableError{IpoStockId}
	}

	OldIpoBid := &IpoBid{}

	if !db.Where("userId = ? AND isClosed = ? AND ipoStockId = ?", UserId, false, IpoStockId).First(&OldIpoBid).RecordNotFound() {
		return 0, IpoOrderStockLimitExceeded{}
	}

	NewIpoBid := &IpoBid{
		UserId:       UserId,
		IpoStockId:   IpoStockId,
		SlotPrice:    SlotPrice,
		SlotQuantity: SlotQuantity,
		CreatedAt:    utils.GetCurrentTimeISO8601(),
		IsFulfilled:  false,
		IsClosed:     false,
	}
	NewIpoBid.UpdatedAt = NewIpoBid.CreatedAt

	price := uint64(SlotQuantity) * SlotPrice

	if err := SaveNewIpoBidTransaction(NewIpoBid, UserId, price, db); err != nil {
		l.Error(err)
		return 0, err
	}

	IpoBidsMap.m[NewIpoBid.Id] = NewIpoBid

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
	if IpoStock1.IsBiddable == false {
		return IpoNotBiddableError{IpoBidToCancel.IpoStockId}
	}

	IpoBidToCancel.IsClosed = true
	IpoBidToCancel.UpdatedAt = utils.GetCurrentTimeISO8601()

	price := uint64(IpoBidToCancel.SlotQuantity) * IpoBidToCancel.SlotPrice

	if err := SaveCancelledIpoBidTransaction(IpoBidToCancel, price, db); err != nil {
		l.Error(err)
		return err
	}

	delete(IpoBidsMap.m, IpoBidId)

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
		return nil, err
	}

	l.Infof("Successfully fetched ipoBid orders for userId : %v", userId)
	return myIpoBids, nil
}

func SaveNewIpoBidTransaction(NewIpoBid *IpoBid, UserId uint32, price uint64, tx *gorm.DB) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "SaveNewIpoBidTransaction",
		"IpoStockId": NewIpoBid.Id,
	})

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

	BiddingUser.Cash -= price
	BiddingUser.ReservedCash += price

	if err := tx.Create(NewIpoBid).Error; err != nil {
		return err
	}

	if err := tx.Save(&BiddingUser).Error; err != nil {
		l.Error(err)
		return err
	}

	return nil
}

func SaveCancelledIpoBidTransaction(IpoBidToCancel *IpoBid, price uint64, tx *gorm.DB) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "SaveCancelledIpoBidTransaction",
		"IpoStockId": IpoBidToCancel.Id,
	})

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

	BiddingUser.Cash += price
	BiddingUser.ReservedCash -= price

	if err := tx.Save(IpoBidToCancel).Error; err != nil {
		return err
	}

	if err := tx.Save(&BiddingUser).Error; err != nil {
		l.Error(err)
		return err
	}

	return nil
}
