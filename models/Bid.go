package models

import (
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"

	"github.com/delta/dalal-street-server/utils"
)

type Bid struct {
	sync.Mutex
	Id                     uint32    `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId                 uint32    `gorm:"column:userId;not null" json:"user_id"`
	StockId                uint32    `gorm:"column:stockId;not null" json:"stock_id"`
	OrderType              OrderType `gorm:"column:orderType;not null" json:"order_type"`
	Price                  uint64    `gorm:"not null" json:"price"`
	StockQuantity          uint64    `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	StockQuantityFulfilled uint64    `gorm:"column:stockQuantityFulFilled;not null"json:"stock_quantity_fulfilled"`
	IsClosed               bool      `gorm:"column:isClosed;not null" json:"is_closed"`
	CreatedAt              string    `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt              string    `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (Bid) TableName() string {
	return "Bids"
}

func (bid *Bid) ToProto() *models_pb.Bid {
	m := make(map[OrderType]models_pb.OrderType)
	m[Limit] = models_pb.OrderType_LIMIT
	m[Market] = models_pb.OrderType_MARKET
	m[StopLoss] = models_pb.OrderType_STOPLOSS

	pBid := &models_pb.Bid{
		Id:                     bid.Id,
		UserId:                 bid.UserId,
		StockId:                bid.StockId,
		Price:                  bid.Price,
		OrderType:              m[bid.OrderType],
		StockQuantity:          bid.StockQuantity,
		StockQuantityFulfilled: bid.StockQuantityFulfilled,
		IsClosed:               bid.IsClosed,
		CreatedAt:              bid.CreatedAt,
		UpdatedAt:              bid.UpdatedAt,
	}

	return pBid
}

// TriggerStoploss will set OrderType to StopLossActive if the ordertype is StopLoss
// Error is returned if that wasn't done successfully.
func (bid *Bid) TriggerStoploss() error {
	var l = logger.WithFields(logrus.Fields{
		"method":      "TriggerStoploss",
		"param_bidId": bid.Id,
	})

	l.Debugf("Attempting")

	db := getDB()
	if bid.OrderType == StopLoss {
		bid.Lock()
		bid.OrderType = StopLossActive
		bid.UpdatedAt = utils.GetCurrentTimeISO8601()
		bid.Unlock()
		if err := db.Save(bid).Error; err != nil {
			l.Errorf("Error while saving data: %+v", err)
			return err
		}
		l.Debugf("Done")
		return nil
	}

	l.Errorf("Called TriggerStoploss on order of type %s", bid.OrderType.String())
	return nil // don't return any error here. Log is sufficient.
}

var bidsMap = struct {
	sync.RWMutex
	m map[uint32]*Bid
}{
	sync.RWMutex{},
	make(map[uint32]*Bid),
}

func getBid(id uint32) (*Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "getBid",
		"param_id": id,
	})

	l.Debugf("Attempting")

	/* Try to see if the bid is there in the map */
	bidsMap.Lock()
	defer bidsMap.Unlock()

	_, ok := bidsMap.m[id]
	if ok {
		l.Debugf("Found bid in bidsMap")
		return bidsMap.m[id], nil
	}

	/* Otherwise load from database */
	l.Debugf("Loading bid from database")
	db := getDB()

	bid := &Bid{}
	db.First(bid, id)

	if bid == nil {
		l.Errorf("Attempted to get non-existing Bid")
		return nil, fmt.Errorf("Bid with id %d does not exist", id)
	}

	bidsMap.m[id] = bid

	l.Debugf("Loaded bid from db: %+v", bid)

	return bid, nil
}

func createBid(bid *Bid) error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "CreateBid",
		"param_bid": fmt.Sprintf("%+v", bid),
	})

	l.Debugf("Attempting")

	db := getDB()

	bid.CreatedAt = utils.GetCurrentTimeISO8601()
	bid.UpdatedAt = bid.CreatedAt

	if err := db.Create(bid).Error; err != nil {
		return err
	}

	bidsMap.Lock()
	bidsMap.m[bid.Id] = bid
	bidsMap.Unlock()

	l.Debugf("Created bid. Id: %d", bid.Id)

	return nil
}

func (bid *Bid) Close() error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "Bid.Close",
		"param_bid": fmt.Sprintf("%+v", bid),
	})

	l.Debugf("Attempting")

	db := getDB()

	bid.Lock()
	if bid.IsClosed {
		bid.Unlock()
		return AlreadyClosedError{bid.Id}
	}
	bid.IsClosed = true
	bid.UpdatedAt = utils.GetCurrentTimeISO8601()
	bid.Unlock()

	if err := db.Save(bid).Error; err != nil {
		l.Error(err)
		return err
	}

	bidsMap.Lock()
	delete(bidsMap.m, bid.Id)
	bidsMap.Unlock()

	l.Debugf("Done")
	return nil
}

// GetAllOpenBids returns all open bids. This will be called by MatchingEngine while initializing.
func GetAllOpenBids() ([]*Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetAllOpenBids",
	})

	l.Infof("Attempting to get all open bid orders")

	db := getDB()

	var openBids []*Bid

	//Load open bid orders from database
	if err := db.Where("isClosed = ?", 0).Find(&openBids).Error; err != nil {
		panic("Error loading open bid orders in matching engine: " + err.Error())
	}

	l.Infof("Done")

	bidsMap.Lock()
	defer bidsMap.Unlock()

	for _, bid := range openBids {
		bidsMap.m[bid.Id] = bid
	}

	return openBids, nil
}

func GetMyOpenBids(userId uint32) ([]*Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyOpenBids",
		"userId": userId,
	})

	l.Infof("Attempting to get open bid orders for userId : %v", userId)

	db := getDB()

	var myOpenBids []*Bid

	if err := db.Where("userId = ? AND isClosed = ?", userId, 0).Find(&myOpenBids).Error; err != nil {
		return nil, err
	}

	l.Infof("Successfully fetched open bid orders for userId : %v", userId)
	return myOpenBids, nil
}

func GetMyClosedBids(userId, lastId, count uint32) (bool, []*Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyClosedBids",
		"userId": userId,
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get closed bid orders for userId : %v", userId)

	db := getDB()

	var myClosedBids []*Bid

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
	if err := db.Where("userId = ? AND isClosed = ?", userId, 1).Order("id desc").Limit(count).Find(&myClosedBids).Error; err != nil {
		return true, nil, err
	}

	var moreExists = len(myClosedBids) >= int(count)
	l.Infof("Successfully fetched closed bid orders for userId : %v", userId)
	return moreExists, myClosedBids, nil
}
