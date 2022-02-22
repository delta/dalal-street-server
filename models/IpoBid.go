package models

import (
	"fmt"
	"sync"

	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/delta/dalal-street-server/utils"
)

// refer models/Bid.go
type IpoBid struct {
	sync.Mutex
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
	sync.RWMutex
	m map[uint32]*IpoBid
}{
	sync.RWMutex{},
	make(map[uint32]*IpoBid),
}

func getIpoBid(id uint32) (*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "getIpoBid",
		"param_id": id,
	})

	l.Debugf("Attempting")

	/* Try to see if the ipoBid is there in the map */
	IpoBidsMap.Lock()
	defer IpoBidsMap.Unlock()

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

func createIpoBid(ipoBid *IpoBid, tx *gorm.DB) error {
	var l = logger.WithFields(logrus.Fields{
		"method":       "CreateIpoBid",
		"param_ipoBid": fmt.Sprintf("%+v", ipoBid),
	})

	l.Debugf("Attempting")

	// ToDo: Add Additional details??
	ipoBid.CreatedAt = utils.GetCurrentTimeISO8601()
	ipoBid.UpdatedAt = ipoBid.CreatedAt

	if err := tx.Create(ipoBid).Error; err != nil {
		return err
	}

	// ToDo: deduct slotprice from user
	IpoBidsMap.Lock()
	IpoBidsMap.m[ipoBid.Id] = ipoBid
	IpoBidsMap.Unlock()

	l.Debugf("Created ipoBid. Id: %d", ipoBid.Id)

	return nil
}

func (ipoBid *IpoBid) Close(tx *gorm.DB) error {
	var l = logger.WithFields(logrus.Fields{
		"method":       "IpoBid.Close",
		"param_ipoBid": fmt.Sprintf("%+v", ipoBid),
	})

	l.Debugf("Attempting")

	ipoBid.Lock()
	if ipoBid.IsClosed {
		ipoBid.Unlock()
		return AlreadyClosedError{ipoBid.Id}
	}
	ipoBid.IsClosed = true
	ipoBid.UpdatedAt = utils.GetCurrentTimeISO8601()
	ipoBid.Unlock()

	if err := tx.Save(ipoBid).Error; err != nil {
		l.Error(err)
		return err
	}

	IpoBidsMap.Lock()
	delete(IpoBidsMap.m, ipoBid.Id)
	IpoBidsMap.Unlock()

	l.Debugf("Done")
	return nil
}

// GetAllOpenIpoBids returns all open ipoBids. This will be called by MatchingEngine while initializing.
func GetAllOpenIpoBids() ([]*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetAllOpenIpoBids",
	})

	l.Infof("Attempting to get all open ipoBid orders")

	db := getDB()

	var openIpoBids []*IpoBid

	//Load open ipoBid orders from database
	if err := db.Where("isClosed = ?", 0).Find(&openIpoBids).Error; err != nil {
		panic("Error loading open ipoBid orders in matching engine: " + err.Error())
	}

	l.Infof("Done")

	IpoBidsMap.Lock()
	defer IpoBidsMap.Unlock()

	for _, ipoBid := range openIpoBids {
		IpoBidsMap.m[ipoBid.Id] = ipoBid
	}

	return openIpoBids, nil
}

func GetAllUnfulfilledIpoBids() ([]*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetAllUnfulfilledIpoBids",
	})

	l.Infof("Attempting to get all unfulfilled ipoBid orders")

	db := getDB()

	var unfulfilledIpoBids []*IpoBid

	//Load open ipoBid orders from database
	if err := db.Where("isClosed = ? AND isFulfilled = ?", 0, 0).Find(&unfulfilledIpoBids).Error; err != nil {
		panic("Error loading open ipoBid orders in matching engine: " + err.Error())
	}

	l.Infof("Done")

	IpoBidsMap.Lock()
	defer IpoBidsMap.Unlock()

	for _, ipoBid := range unfulfilledIpoBids {
		IpoBidsMap.m[ipoBid.Id] = ipoBid
	}

	return unfulfilledIpoBids, nil
}

func GetAllFulfilledIpoBids() ([]*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetAllFulfilledIpoBids",
	})

	l.Infof("Attempting to get all fulfilled ipoBid orders")

	db := getDB()

	var fulfilledIpoBids []*IpoBid

	//Load open ipoBid orders from database
	if err := db.Where("isClosed = ? AND isFulfilled = ?", 0, 1).Find(&fulfilledIpoBids).Error; err != nil {
		panic("Error loading open ipoBid orders in matching engine: " + err.Error())
	}

	l.Infof("Done")

	IpoBidsMap.Lock()
	defer IpoBidsMap.Unlock()

	for _, ipoBid := range fulfilledIpoBids {
		IpoBidsMap.m[ipoBid.Id] = ipoBid
	}

	return fulfilledIpoBids, nil
}

// Combine open and closed ipoBids into a single function??
func GetMyOpenIpoBids(userId uint32) ([]*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyOpenIpoBids",
		"userId": userId,
	})

	l.Infof("Attempting to get open ipoBid orders for userId : %v", userId)

	db := getDB()

	var myOpenIpoBids []*IpoBid

	if err := db.Where("userId = ? AND isClosed = ?", userId, 0).Find(&myOpenIpoBids).Error; err != nil {
		return nil, err
	}

	l.Infof("Successfully fetched open ipoBid orders for userId : %v", userId)
	return myOpenIpoBids, nil
}

func GetMyClosedIpoBids(userId, lastId, count uint32) (bool, []*IpoBid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyClosedIpoBids",
		"userId": userId,
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get closed ipoBid orders for userId : %v", userId)

	db := getDB()

	var myClosedIpoBids []*IpoBid

	//set default value of count if it is zero
	if count == 0 {
		count = MY_BID_COUNT
	} else {
		count = utils.MinInt32(count, MY_BID_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	if err := db.Where("userId = ? AND isClosed = ?", userId, 1).Order("id desc").Limit(count).Find(&myClosedIpoBids).Error; err != nil {
		return true, nil, err
	}

	var moreExists = len(myClosedIpoBids) >= int(count)
	l.Infof("Successfully fetched closed ipoBid orders for userId : %v", userId)
	return moreExists, myClosedIpoBids, nil
}
