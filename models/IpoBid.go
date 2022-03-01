package models

import (
	"fmt"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/sirupsen/logrus"

	"github.com/delta/dalal-street-server/utils"
)

// refer models/Bid.go
type IpoBid struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId       uint32 `gorm:"column:userId;not null" json:"user_id"`
	IpoStockId   uint32 `gorm:"column:ipoStockId;not null" json:"ipo_stock_id"`
	SlotPrice    uint64 `gorm:"not null" json:"slot_price"`
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

// to prevent people from bidding on ipo stocks after window closes (like days 4-5),
// should we create a new field "isBiddable" in IpoStock, or just delete the IpoStock
// from the ipostocks table after alloting it ?

func CreateIpoBid(UserId uint32, IpoStockId uint32, SlotQuantity uint32, SlotPrice uint64) (uint32, error) {

	var l = logger.WithFields(logrus.Fields{
		"method":             "CreateIpoBid",
		"param_userId":       fmt.Sprintf("%+v", UserId),
		"param_ipoStockId":   fmt.Sprintf("%+v", IpoStockId),
		"param_slotQuantity": fmt.Sprintf("%+v", SlotQuantity),
	})

	l.Debugf("Attempting")

	if SlotQuantity > 1 {
		return 0, OrderStockLimitExceeded{}
	}
	db := getDB()
	BiddingUser := &User{}
	db.First(BiddingUser, UserId)

	if BiddingUser.Cash < SlotPrice {
		return 0, NotEnoughCashError{}
	}

	NewIpoBid := &IpoBid{
		UserId:       UserId,
		IpoStockId:   IpoStockId,
		SlotPrice:    SlotPrice,
		SlotQuantity: SlotQuantity,
		IsFulfilled:  false,
		IsClosed:     false,
	}

	NewIpoBid.CreatedAt = utils.GetCurrentTimeISO8601()
	NewIpoBid.UpdatedAt = NewIpoBid.CreatedAt

	price := uint64(SlotQuantity) * SlotPrice

	if BiddingUser == nil {
		l.Errorf("non-existant user tried to create an IpoBid")
		return 0, fmt.Errorf("Ipo BiddingUser with id %d does not exist", UserId)
	}

	// // Lock doesnt exist for BiddingUser -- how to ensure
	// // that values are saved correctly without external interference ?
	// BiddingUser.lock
	BiddingUser.Cash = BiddingUser.Cash - price
	BiddingUser.ReservedCash = BiddingUser.ReservedCash + price
	// BiddingUser.unlock

	if err := db.Create(NewIpoBid).Error; err != nil {
		return 0, err
	}

	IpoBidsMap.m[NewIpoBid.Id] = NewIpoBid

	l.Debugf("Created ipoBid. Id: %d", NewIpoBid.Id)
	// Will NewIpoBid.Id even be defined since MYSQL determines it??

	return NewIpoBid.Id, nil
}

func CancelIpoBid(IpoBidId uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":         "IpoBid.Cancel",
		"param_ipoBidId": fmt.Sprintf("%+v", IpoBidId),
	})

	l.Debugf("Attempting")

	db := getDB()

	var IpoBidToCancel IpoBid

	if err := db.First(IpoBidToCancel, IpoBidId).Error; err != nil {
		return err
	}

	if IpoBidToCancel.IsClosed {
		return AlreadyClosedError{IpoBidToCancel.Id}
	}

	IpoBidToCancel.IsClosed = true
	IpoBidToCancel.UpdatedAt = utils.GetCurrentTimeISO8601()

	price := uint64(IpoBidToCancel.SlotQuantity) * IpoBidToCancel.SlotPrice

	CancelBidUser := &User{}
	db.First(CancelBidUser, IpoBidToCancel.UserId)

	if CancelBidUser == nil {
		l.Errorf("non-existent user tried to cancel an IpoBid")
		return fmt.Errorf("User with id %d does not exist", IpoBidToCancel.UserId)
	}

	// // Lock doesnt exist for CancelBidUser
	// CancelBidUser.lock
	CancelBidUser.Cash += price
	CancelBidUser.ReservedCash -= price
	// CancelBidUser.unlock

	if err := db.Save(IpoBidToCancel).Error; err != nil {
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
