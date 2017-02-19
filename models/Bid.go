package models

import (
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"

	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type Bid struct {
	Id                     uint32    `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId                 uint32    `gorm:"column:userId;not null" json:"user_id"`
	StockId                uint32    `gorm:"column:stockId;not null" json:"stock_id"`
	OrderType              OrderType `gorm:"column:orderType;not null" json:"order_type"`
	Price                  uint32    `gorm:"not null" json:"price"`
	StockQuantity          uint32    `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	StockQuantityFulfilled uint32    `gorm:"column:stockQuantityFulFilled;not null"json:"stock_quantity_fulfilled"`
	IsClosed               bool      `gorm:"column:isClosed;not null" json:"is_closed"`
	CreatedAt              string    `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt              string    `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (Bid) TableName() string {
	return "Bids"
}

func (gBid *Bid) ToProto() *models_proto.Bid {
	pBid := &models_proto.Bid{
		Id:      gBid.Id,
		UserId:  gBid.UserId,
		StockId: gBid.StockId,
		Price:   gBid.Price,
		//	OrderType              OrderType `protobuf:"varint,5,opt,name=order_type,json=orderType,enum=dalalstreet.socketapi.models.OrderType" json:"order_type,omitempty"`
		StockQuantity:          gBid.StockQuantity,
		StockQuantityFulfilled: gBid.StockQuantityFulfilled,
		IsClosed:               gBid.IsClosed,
		CreatedAt:              gBid.CreatedAt,
		UpdatedAt:              gBid.UpdatedAt,
	}
	if gBid.OrderType == Limit {
		pBid.OrderType = models_proto.OrderType_LIMIT
	} else if gBid.OrderType == Market {
		pBid.OrderType = models_proto.OrderType_MARKET
	} else if gBid.OrderType == Stoploss {
		pBid.OrderType = models_proto.OrderType_STOPLOSS
	}
	return pBid
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
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	bidsMap.m[id] = &Bid{}
	bid := bidsMap.m[id]
	db.First(bid, id)

	if bid == nil {
		l.Errorf("Attempted to get non-existing Bid")
		return nil, fmt.Errorf("Bid with id %d does not exist", id)
	}

	l.Debugf("Loaded bid from db: %+v", bid)

	return bid, nil
}

/*
func getBidCopy(id uint32) (chan struct{}, *Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "getBidCopy",
		"param_id": id,
	})

	var (
		a *bidAndLock
		ch = make(chan struct{})
	)

	l.Debugf("Attempting")

	/* Try to see if the bid is there in the map * /
	bidLocks.RLock()
	a, ok := bidLocks.m[id]
	bidLocks.Unlock()
	if ok {
		l.Debugf("Found bid in bidLocks map. Locking.")
		a.Lock()
		go func() {
			l.Debugf("Waiting for caller to release lock")
			<-ch
			a.Unlock()
			l.Debugf("Lock released")
		}()
		return ch, a.bid, nil
	}

	/* Otherwise load from database * /
	l.Debugf("Loading bid from database")
	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, nil, err
	}
	defer db.Close()

	bidLocks.Lock()
	db.First(a.bid, id)
	bidLocks.Unlock()

	if a.bid == nil {
		l.Errorf("Attempted to get non-existing Bid")
		return nil, nil, fmt.Errorf("Bid with id %d does not exist", id)
	}

	l.Debugf("Loaded bid from db. Locking")
	a.RLock()
	go func() {
		l.Debugf("Waiting for caller to release lock")
		<-ch
		bidLocks.m[id].Unlock()
		l.Debugf("Lock released")
	}()

	l.Debugf("Bid: %+v", a.bid)

	return ch, a.bid, nil
}
*/
func createBid(bid *Bid) error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "CreateBid",
		"param_bid": fmt.Sprintf("%+v", bid),
	})

	l.Debugf("Attempting")

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err
	}
	defer db.Close()

	if err := db.Create(bid).Error; err != nil {
		return err
	}

	l.Debugf("Created bid. Id: %d", bid.Id)

	return nil
}

func (bid *Bid) Close() error {
	var l = logger.WithFields(logrus.Fields{
		"method":    "Bid.Close",
		"param_bid": fmt.Sprintf("%+v", bid),
	})

	l.Debugf("Attempting")

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err
	}
	defer db.Close()
	bid.IsClosed = true

	if err := db.Save(bid).Error; err != nil {
		l.Error(err)
		return err
	}
	l.Debugf("Done")
	return nil
}

func GetMyBids(userId, lastId, count uint32) (bool, map[uint32]*Bid, map[uint32]*Bid, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetMyBids",
		"userId": userId,
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get bid orders for userId : %v", userId)

	db, err := DbOpen()
	if err != nil {
		return true, nil, nil, err
	}
	defer db.Close()

	var myOpenBids []*Bid
	var myClosedBids []*Bid

	//get all open bids
	if err := db.Where("userId = ? and isClosed = ?", userId, 0).Find(&myOpenBids).Error; err != nil {
		return true, nil, nil, err
	}

	//set default value of count if it is zero
	if count == 0 {
		count = MY_BID_COUNT
	} else {
		count = min(count, MY_BID_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	if err := db.Where("userId = ?", userId).Order("desc id").Limit(count).Find(&myClosedBids).Error; err != nil {
		return true, nil, nil, err
	}

	myOpenBidsMap := make(map[uint32]*Bid)

	for _, bid := range myOpenBids {
		myOpenBidsMap[bid.Id] = bid
	}

	myClosedBidsMap := make(map[uint32]*Bid)

	for _, bid := range myClosedBids {
		myClosedBidsMap[bid.Id] = bid
	}

	var moreExists = len(myClosedBids) >= int(count)
	l.Infof("Successfully fetched bid orders for userId : %v", userId)
	return moreExists, myOpenBidsMap, myClosedBidsMap, nil
}
